// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kubernetes

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/transport/spdy"

	"github.com/elastic/beats/v7/dev-tools/mage"
)

const sshBitSize = 4096

var mode = int32(256)

// KubeRemote rsyncs the passed directory to a pod and runs the command inside of that pod.
type KubeRemote struct {
	cfg       *rest.Config
	cs        *kubernetes.Clientset
	namespace string
	name      string
	workDir   string
	destDir   string
	syncDir   string

	svcAccName string
	secretName string
	privateKey []byte
	publicKey  []byte
}

// NewKubeRemote creates a new kubernetes remote runner.
func NewKubeRemote(kubeconfig string, namespace string, name string, workDir string, destDir string, syncDir string) (*KubeRemote, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	name = strings.Replace(name, "_", "-", -1)
	svcAccName := fmt.Sprintf("%s-sa", name)
	secretName := fmt.Sprintf("%s-ssh-key", name)
	privateKey, publicKey, err := generateSSHKeyPair()
	if err != nil {
		return nil, err
	}
	return &KubeRemote{config, cs, namespace, name, workDir, destDir, syncDir, svcAccName, secretName, privateKey, publicKey}, nil
}

// Run runs the command remotely on the kubernetes cluster.
func (r *KubeRemote) Run(env map[string]string, stdout io.Writer, stderr io.Writer, args ...string) error {
	if err := r.syncSSHKey(); err != nil {
		return fmt.Errorf("failed to sync SSH secret: %w", err)
	}
	defer r.deleteSSHKey()
	if err := r.syncServiceAccount(); err != nil {
		return err
	}
	defer r.deleteServiceAccount()
	_, err := r.createPod(env, args...)
	if err != nil {
		return fmt.Errorf("failed to create execute pod: %w", err)
	}
	defer r.deletePod()

	// wait for SSH to be up inside the init container.
	_, err = r.waitForPod(5*time.Minute, podInitReady)
	if err != nil {
		return fmt.Errorf("execute pod init container never started: %w", err)
	}
	time.Sleep(1 * time.Second) // SSH inside of container can take a moment

	// forward the SSH port so rsync can be ran.
	randomPort, err := getFreePort()
	if err != nil {
		return fmt.Errorf("failed to find a free port: %w", err)
	}
	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{}, 1)
	f, err := r.portForward([]string{fmt.Sprintf("%d:%d", randomPort, 22)}, stopChannel, readyChannel, stderr, stderr)
	if err != nil {
		return err
	}
	go f.ForwardPorts()
	<-readyChannel

	// perform the rsync
	r.rsync(randomPort, stderr, stderr)

	// stop port forwarding
	close(stopChannel)

	// wait for exec container to be running
	_, err = r.waitForPod(5*time.Minute, containerRunning("exec"))
	if err != nil {
		return fmt.Errorf("execute pod container never started: %w", err)
	}

	// stream the logs of the container
	err = r.streamLogs("exec", stdout)
	if err != nil {
		return fmt.Errorf("failed to stream the logs: %w", err)
	}

	// wait for exec container to be completely done
	pod, err := r.waitForPod(30*time.Second, podDone)
	if err != nil {
		return fmt.Errorf("execute pod didn't terminate after 30 seconds of log stream: %w", err)
	}

	// return error on failure
	if pod.Status.Phase == apiv1.PodFailed {
		return fmt.Errorf("execute pod test failed")
	}
	return nil
}

// deleteSSHKey deletes SSH key from the cluster.
func (r *KubeRemote) deleteSSHKey() {
	_ = r.cs.CoreV1().Secrets(r.namespace).Delete(context.TODO(), r.secretName, metav1.DeleteOptions{})
}

// syncSSHKey syncs the SSH key to the cluster.
func (r *KubeRemote) syncSSHKey() error {
	// delete before create
	r.deleteSSHKey()
	_, err := r.cs.CoreV1().Secrets(r.namespace).Create(
		context.TODO(),
		createSecretManifest(r.secretName, r.publicKey),
		metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// deleteServiceAccount syncs required service account.
func (r *KubeRemote) deleteServiceAccount() {
	ctx := context.TODO()
	_ = r.cs.RbacV1().ClusterRoleBindings().Delete(ctx, r.name, metav1.DeleteOptions{})
	_ = r.cs.RbacV1().ClusterRoles().Delete(ctx, r.name, metav1.DeleteOptions{})
	_ = r.cs.CoreV1().ServiceAccounts(r.namespace).Delete(ctx, r.svcAccName, metav1.DeleteOptions{})
}

// syncServiceAccount syncs required service account.
func (r *KubeRemote) syncServiceAccount() error {
	ctx := context.TODO()
	// delete before create
	r.deleteServiceAccount()
	_, err := r.cs.CoreV1().ServiceAccounts(r.namespace).Create(
		ctx,
		createServiceAccountManifest(r.svcAccName),
		metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}
	_, err = r.cs.RbacV1().ClusterRoles().Create(ctx, createClusterRoleManifest(r.name), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}
	_, err = r.cs.RbacV1().ClusterRoleBindings().Create(
		ctx,
		createClusterRoleBindingManifest(r.name, r.namespace, r.svcAccName),
		metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}
	return nil
}

// createPod creates the pod.
func (r *KubeRemote) createPod(env map[string]string, cmd ...string) (*apiv1.Pod, error) {
	version, err := mage.GoVersion()
	if err != nil {
		return nil, err
	}
	image := fmt.Sprintf("golang:%s", version)
	r.deletePod() // ensure it doesn't already exist
	return r.cs.CoreV1().Pods(r.namespace).Create(
		context.TODO(),
		createPodManifest(r.name, image, env, cmd, r.workDir, r.destDir, r.secretName, r.svcAccName),
		metav1.CreateOptions{})
}

// deletePod deletes the pod.
func (r *KubeRemote) deletePod() {
	_ = r.cs.CoreV1().Pods(r.namespace).Delete(context.TODO(), r.name, metav1.DeleteOptions{})
}

// waitForPod waits for the created pod to match the given condition.
func (r *KubeRemote) waitForPod(wait time.Duration, condition watchtools.ConditionFunc) (*apiv1.Pod, error) {
	w, err := r.cs.CoreV1().Pods(r.namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: r.name}))
	if err != nil {
		return nil, err
	}

	ctx, _ := watchtools.ContextWithOptionalTimeout(context.Background(), wait)
	ev, err := watchtools.UntilWithoutRetry(ctx, w, func(ev watch.Event) (bool, error) {
		return condition(ev)
	})
	if ev != nil {
		return ev.Object.(*apiv1.Pod), err
	}
	return nil, err
}

// portForward runs the port forwarding so SSH rsync can be ran into the pod.
func (r *KubeRemote) portForward(ports []string, stopChannel, readyChannel chan struct{}, stdout, stderr io.Writer) (*portforward.PortForwarder, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(r.cfg)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", r.namespace, r.name)
	hostIP := strings.TrimLeft(r.cfg.Host, "https://")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)
	return portforward.New(dialer, ports, stopChannel, readyChannel, stdout, stderr)
}

// rsync performs the rsync of sync directory to destination directory inside of the pod.
func (r *KubeRemote) rsync(port uint16, stdout, stderr io.Writer) error {
	privateKeyFile, err := createTempFile(r.privateKey)
	if err != nil {
		return err
	}

	rsh := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p %d -i %s", port, privateKeyFile)
	args := []string{
		"--rsh", rsh,
		"-a", fmt.Sprintf("%s/", r.syncDir),
		fmt.Sprintf("root@localhost:%s", r.destDir),
	}
	cmd := exec.Command("rsync", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// streamLogs streams the logs from the execution pod until the pod is terminated.
func (r *KubeRemote) streamLogs(container string, stdout io.Writer) error {
	req := r.cs.CoreV1().Pods(r.namespace).GetLogs(r.name, &apiv1.PodLogOptions{
		Container: container,
		Follow:    true,
	})
	logs, err := req.Stream(context.TODO())
	if err != nil {
		return err
	}
	defer logs.Close()

	reader := bufio.NewReader(logs)
	for {
		bytes, err := reader.ReadBytes('\n')
		if _, err := stdout.Write(bytes); err != nil {
			return err
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
	}
}

// generateSSHKeyPair generates a new SSH key pair.
func generateSSHKeyPair() ([]byte, []byte, error) {
	private, err := rsa.GenerateKey(rand.Reader, sshBitSize)
	if err != nil {
		return nil, nil, err
	}
	if err = private.Validate(); err != nil {
		return nil, nil, err
	}
	public, err := ssh.NewPublicKey(&private.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	return encodePrivateKeyToPEM(private), ssh.MarshalAuthorizedKey(public), nil
}

// encodePrivateKeyToPEM encodes private key from RSA to PEM format.
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}
	return pem.EncodeToMemory(&privBlock)
}

// getFreePort finds a free port.
func getFreePort() (uint16, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port), nil
}

// createSecretManifest creates the secret object to create in the cluster.
//
// This is the public key that the sshd uses as the authorized key.
func createSecretManifest(name string, publicKey []byte) *apiv1.Secret {
	return &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: map[string]string{
			"authorized_keys": string(publicKey),
		},
	}
}

// createServiceAccountManifest creates the service account the pod will used.
func createServiceAccountManifest(name string) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// createClusterRoleManifest creates the cluster role the pod will used.
//
// This gives the pod all permissions on everything!
func createClusterRoleManifest(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
			rbacv1.PolicyRule{
				Verbs:           []string{"*"},
				NonResourceURLs: []string{"*"},
			},
		},
	}
}

// createClusterRoleBindingManifest creates the cluster role binding the pod will used.
//
// This binds the service account to the cluster role.
func createClusterRoleBindingManifest(name string, namespace string, svcAccName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      svcAccName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
	}
}

// createPodManifest creates the pod inside of the cluster that will be used for remote execution.
//
// Creates a pod with an init container that runs sshd-rsync, once the first connection closes the init container
// exits then the exec container starts using the rsync'd directory as its work directory.
func createPodManifest(name string, image string, env map[string]string, cmd []string, workDir string, destDir string, secretName string, svcAccName string) *apiv1.Pod {
	execEnv := []apiv1.EnvVar{
		apiv1.EnvVar{
			Name: "NODE_NAME",
			ValueFrom: &apiv1.EnvVarSource{
				FieldRef: &apiv1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}
	for k, v := range env {
		execEnv = append(execEnv, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.PodSpec{
			ServiceAccountName: svcAccName,
			HostNetwork:        true,
			DNSPolicy:          apiv1.DNSClusterFirstWithHostNet,
			RestartPolicy:      apiv1.RestartPolicyNever,
			InitContainers: []apiv1.Container{
				{
					Name:  "sync-init",
					Image: "ernoaapa/sshd-rsync",
					Ports: []apiv1.ContainerPort{
						{
							Name:          "ssh",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: 22,
						},
					},
					Env: []apiv1.EnvVar{
						{
							Name:  "ONE_TIME",
							Value: "true",
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "ssh-config",
							MountPath: "/root/.ssh/authorized_keys",
							SubPath:   "authorized_keys",
						},
						{
							Name:      "destdir",
							MountPath: destDir,
						},
					},
				},
			},
			Containers: []apiv1.Container{
				{
					Name:       "exec",
					Image:      image,
					Command:    cmd,
					WorkingDir: workDir,
					Env:        execEnv,
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "destdir",
							MountPath: destDir,
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "ssh-config",
					VolumeSource: apiv1.VolumeSource{
						Secret: &apiv1.SecretVolumeSource{
							SecretName:  secretName,
							DefaultMode: &mode,
						},
					},
				},
				{
					Name: "destdir",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func podInitReady(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}
	switch t := event.Object.(type) {
	case *apiv1.Pod:
		switch t.Status.Phase {
		case apiv1.PodFailed, apiv1.PodSucceeded:
			return false, nil
		case apiv1.PodRunning:
			return false, nil
		case apiv1.PodPending:
			return isInitContainersReady(t), nil
		}
	}
	return false, nil
}

func isInitContainersReady(pod *apiv1.Pod) bool {
	if isScheduled(pod) && isInitContainersRunning(pod) {
		return true
	}
	return false
}

func isScheduled(pod *apiv1.Pod) bool {
	if &pod.Status != nil && len(pod.Status.Conditions) > 0 {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == apiv1.PodScheduled &&
				condition.Status == apiv1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func isInitContainersRunning(pod *apiv1.Pod) bool {
	if &pod.Status != nil {
		if len(pod.Spec.InitContainers) != len(pod.Status.InitContainerStatuses) {
			return false
		}
		for _, status := range pod.Status.InitContainerStatuses {
			if status.State.Running == nil {
				return false
			}
		}
		return true
	}
	return false
}

func containerRunning(containerName string) func(watch.Event) (bool, error) {
	return func(event watch.Event) (bool, error) {
		switch event.Type {
		case watch.Deleted:
			return false, k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
		}
		switch t := event.Object.(type) {
		case *apiv1.Pod:
			switch t.Status.Phase {
			case apiv1.PodFailed, apiv1.PodSucceeded:
				return false, nil
			case apiv1.PodRunning:
				return isContainerRunning(t, containerName)
			}
		}
		return false, nil
	}
}

func isContainerRunning(pod *apiv1.Pod, containerName string) (bool, error) {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			if status.State.Waiting != nil {
				return false, nil
			} else if status.State.Running != nil {
				return true, nil
			} else if status.State.Terminated != nil {
				return false, nil
			} else {
				return false, fmt.Errorf("Unknown container state")
			}
		}
	}
	return false, nil
}

func podDone(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}
	switch t := event.Object.(type) {
	case *apiv1.Pod:
		switch t.Status.Phase {
		case apiv1.PodFailed, apiv1.PodSucceeded:
			return true, nil
		}
	}
	return false, nil
}

func createTempFile(content []byte) (string, error) {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	tmpfile, err := ioutil.TempFile("", hex.EncodeToString(randBytes))
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	if _, err := tmpfile.Write(content); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}
