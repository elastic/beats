# generating certificate (provide 'capass' when asked for password):
openssl req -x509 -key ca.key -out ca.cer -days 365000 -subj "/CN=localhost" -addext "subjectAltName = DNS:localhost"
