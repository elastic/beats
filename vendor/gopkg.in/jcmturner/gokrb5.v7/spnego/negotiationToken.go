package spnego

import (
	"context"
	"errors"
	"fmt"

	"github.com/jcmturner/gofork/encoding/asn1"
	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/gssapi"
	"gopkg.in/jcmturner/gokrb5.v7/messages"
	"gopkg.in/jcmturner/gokrb5.v7/service"
	"gopkg.in/jcmturner/gokrb5.v7/types"
)

/*
https://msdn.microsoft.com/en-us/library/ms995330.aspx

NegotiationToken ::= CHOICE {
  negTokenInit    [0] NegTokenInit,  This is the Negotiation token sent from the client to the server.
  negTokenResp    [1] NegTokenResp
}

NegTokenInit ::= SEQUENCE {
  mechTypes       [0] MechTypeList,
  reqFlags        [1] ContextFlags  OPTIONAL,
  -- inherited from RFC 2478 for backward compatibility,
  -- RECOMMENDED to be left out
  mechToken       [2] OCTET STRING  OPTIONAL,
  mechListMIC     [3] OCTET STRING  OPTIONAL,
  ...
}

NegTokenResp ::= SEQUENCE {
  negState       [0] ENUMERATED {
    accept-completed    (0),
    accept-incomplete   (1),
    reject              (2),
    request-mic         (3)
  }                                 OPTIONAL,
  -- REQUIRED in the first reply from the target
  supportedMech   [1] MechType      OPTIONAL,
  -- present only in the first reply from the target
  responseToken   [2] OCTET STRING  OPTIONAL,
  mechListMIC     [3] OCTET STRING  OPTIONAL,
  ...
}
*/

// Negotiation state values.
const (
	NegStateAcceptCompleted  NegState = 0
	NegStateAcceptIncomplete NegState = 1
	NegStateReject           NegState = 2
	NegStateRequestMIC       NegState = 3
)

// NegState is a type to indicate the SPNEGO negotiation state.
type NegState int

// NegTokenInit implements Negotiation Token of type Init.
type NegTokenInit struct {
	MechTypes      []asn1.ObjectIdentifier
	ReqFlags       gssapi.ContextFlags
	MechTokenBytes []byte
	MechListMIC    []byte
	mechToken      gssapi.ContextToken
	settings       *service.Settings
}

type marshalNegTokenInit struct {
	MechTypes      []asn1.ObjectIdentifier `asn1:"explicit,tag:0"`
	ReqFlags       gssapi.ContextFlags     `asn1:"explicit,optional,tag:1"`
	MechTokenBytes []byte                  `asn1:"explicit,optional,omitempty,tag:2"`
	MechListMIC    []byte                  `asn1:"explicit,optional,omitempty,tag:3"` // This field is not used when negotiating Kerberos tokens
}

// NegTokenResp implements Negotiation Token of type Resp/Targ
type NegTokenResp struct {
	NegState      asn1.Enumerated
	SupportedMech asn1.ObjectIdentifier
	ResponseToken []byte
	MechListMIC   []byte
	mechToken     gssapi.ContextToken
	settings      *service.Settings
}

type marshalNegTokenResp struct {
	NegState      asn1.Enumerated       `asn1:"explicit,tag:0"`
	SupportedMech asn1.ObjectIdentifier `asn1:"explicit,optional,tag:1"`
	ResponseToken []byte                `asn1:"explicit,optional,omitempty,tag:2"`
	MechListMIC   []byte                `asn1:"explicit,optional,omitempty,tag:3"` // This field is not used when negotiating Kerberos tokens
}

// NegTokenTarg implements Negotiation Token of type Resp/Targ
type NegTokenTarg NegTokenResp

// Marshal an Init negotiation token
func (n *NegTokenInit) Marshal() ([]byte, error) {
	m := marshalNegTokenInit{
		MechTypes:      n.MechTypes,
		ReqFlags:       n.ReqFlags,
		MechTokenBytes: n.MechTokenBytes,
		MechListMIC:    n.MechListMIC,
	}
	b, err := asn1.Marshal(m)
	if err != nil {
		return nil, err
	}
	nt := asn1.RawValue{
		Tag:        0,
		Class:      2,
		IsCompound: true,
		Bytes:      b,
	}
	nb, err := asn1.Marshal(nt)
	if err != nil {
		return nil, err
	}
	return nb, nil
}

// Unmarshal an Init negotiation token
func (n *NegTokenInit) Unmarshal(b []byte) error {
	init, nt, err := UnmarshalNegToken(b)
	if err != nil {
		return err
	}
	if !init {
		return errors.New("bytes were not that of a NegTokenInit")
	}
	nInit := nt.(NegTokenInit)
	n.MechTokenBytes = nInit.MechTokenBytes
	n.MechListMIC = nInit.MechListMIC
	n.MechTypes = nInit.MechTypes
	n.ReqFlags = nInit.ReqFlags
	return nil
}

// Verify an Init negotiation token
func (n *NegTokenInit) Verify() (bool, gssapi.Status) {
	// Check if supported mechanisms are in the MechTypeList
	var mtSupported bool
	for _, m := range n.MechTypes {
		if m.Equal(gssapi.OID(gssapi.OIDKRB5)) || m.Equal(gssapi.OID(gssapi.OIDMSLegacyKRB5)) {
			if n.mechToken == nil && n.MechTokenBytes == nil {
				return false, gssapi.Status{Code: gssapi.StatusContinueNeeded}
			}
			mtSupported = true
			break
		}
	}
	if !mtSupported {
		return false, gssapi.Status{Code: gssapi.StatusBadMech, Message: "no supported mechanism specified in negotiation"}
	}
	// There should be some mechtoken bytes for a KRB5Token (other mech types are not supported)
	mt := new(KRB5Token)
	mt.settings = n.settings
	if n.mechToken == nil {
		err := mt.Unmarshal(n.MechTokenBytes)
		if err != nil {
			return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: err.Error()}
		}
		n.mechToken = mt
	} else {
		var ok bool
		mt, ok = n.mechToken.(*KRB5Token)
		if !ok {
			return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: "MechToken is not a KRB5 token as expected"}
		}
	}
	// Verify the mechtoken
	return n.mechToken.Verify()
}

// Context returns the SPNEGO context which will contain any verify user identity information.
func (n *NegTokenInit) Context() context.Context {
	if n.mechToken != nil {
		mt, ok := n.mechToken.(*KRB5Token)
		if !ok {
			return nil
		}
		return mt.Context()
	}
	return nil
}

// Marshal a Resp/Targ negotiation token
func (n *NegTokenResp) Marshal() ([]byte, error) {
	m := marshalNegTokenResp{
		NegState:      n.NegState,
		SupportedMech: n.SupportedMech,
		ResponseToken: n.ResponseToken,
		MechListMIC:   n.MechListMIC,
	}
	b, err := asn1.Marshal(m)
	if err != nil {
		return nil, err
	}
	nt := asn1.RawValue{
		Tag:        1,
		Class:      2,
		IsCompound: true,
		Bytes:      b,
	}
	nb, err := asn1.Marshal(nt)
	if err != nil {
		return nil, err
	}
	return nb, nil
}

// Unmarshal a Resp/Targ negotiation token
func (n *NegTokenResp) Unmarshal(b []byte) error {
	init, nt, err := UnmarshalNegToken(b)
	if err != nil {
		return err
	}
	if init {
		return errors.New("bytes were not that of a NegTokenResp")
	}
	nResp := nt.(NegTokenResp)
	n.MechListMIC = nResp.MechListMIC
	n.NegState = nResp.NegState
	n.ResponseToken = nResp.ResponseToken
	n.SupportedMech = nResp.SupportedMech
	return nil
}

// Verify a Resp/Targ negotiation token
func (n *NegTokenResp) Verify() (bool, gssapi.Status) {
	if n.SupportedMech.Equal(gssapi.OID(gssapi.OIDKRB5)) || n.SupportedMech.Equal(gssapi.OID(gssapi.OIDMSLegacyKRB5)) {
		if n.mechToken == nil && n.ResponseToken == nil {
			return false, gssapi.Status{Code: gssapi.StatusContinueNeeded}
		}
		mt := new(KRB5Token)
		mt.settings = n.settings
		if n.mechToken == nil {
			err := mt.Unmarshal(n.ResponseToken)
			if err != nil {
				return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: err.Error()}
			}
			n.mechToken = mt
		} else {
			var ok bool
			mt, ok = n.mechToken.(*KRB5Token)
			if !ok {
				return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: "MechToken is not a KRB5 token as expected"}
			}
		}
		if mt == nil {
			return false, gssapi.Status{Code: gssapi.StatusContinueNeeded}
		}
		// Verify the mechtoken
		return mt.Verify()
	}
	return false, gssapi.Status{Code: gssapi.StatusBadMech, Message: "no supported mechanism specified in negotiation"}
}

// State returns the negotiation state of the negotiation response.
func (n *NegTokenResp) State() NegState {
	return NegState(n.NegState)
}

// Context returns the SPNEGO context which will contain any verify user identity information.
func (n *NegTokenResp) Context() context.Context {
	if n.mechToken != nil {
		mt, ok := n.mechToken.(*KRB5Token)
		if !ok {
			return nil
		}
		return mt.Context()
	}
	return nil
}

// UnmarshalNegToken umarshals and returns either a NegTokenInit or a NegTokenResp.
//
// The boolean indicates if the response is a NegTokenInit.
// If error is nil and the boolean is false the response is a NegTokenResp.
func UnmarshalNegToken(b []byte) (bool, interface{}, error) {
	var a asn1.RawValue
	_, err := asn1.Unmarshal(b, &a)
	if err != nil {
		return false, nil, fmt.Errorf("error unmarshalling NegotiationToken: %v", err)
	}
	switch a.Tag {
	case 0:
		var n marshalNegTokenInit
		_, err = asn1.Unmarshal(a.Bytes, &n)
		if err != nil {
			return false, nil, fmt.Errorf("error unmarshalling NegotiationToken type %d (Init): %v", a.Tag, err)
		}
		nt := NegTokenInit{
			MechTypes:      n.MechTypes,
			ReqFlags:       n.ReqFlags,
			MechTokenBytes: n.MechTokenBytes,
			MechListMIC:    n.MechListMIC,
		}
		return true, nt, nil
	case 1:
		var n marshalNegTokenResp
		_, err = asn1.Unmarshal(a.Bytes, &n)
		if err != nil {
			return false, nil, fmt.Errorf("error unmarshalling NegotiationToken type %d (Resp/Targ): %v", a.Tag, err)
		}
		nt := NegTokenResp{
			NegState:      n.NegState,
			SupportedMech: n.SupportedMech,
			ResponseToken: n.ResponseToken,
			MechListMIC:   n.MechListMIC,
		}
		return false, nt, nil
	default:
		return false, nil, errors.New("unknown choice type for NegotiationToken")
	}

}

// NewNegTokenInitKRB5 creates new Init negotiation token for Kerberos 5
func NewNegTokenInitKRB5(cl *client.Client, tkt messages.Ticket, sessionKey types.EncryptionKey) (NegTokenInit, error) {
	mt, err := NewKRB5TokenAPREQ(cl, tkt, sessionKey, []int{gssapi.ContextFlagInteg, gssapi.ContextFlagConf}, []int{})
	if err != nil {
		return NegTokenInit{}, fmt.Errorf("error getting KRB5 token; %v", err)
	}
	mtb, err := mt.Marshal()
	if err != nil {
		return NegTokenInit{}, fmt.Errorf("error marshalling KRB5 token; %v", err)
	}
	return NegTokenInit{
		MechTypes:      []asn1.ObjectIdentifier{gssapi.OID(gssapi.OIDKRB5)},
		MechTokenBytes: mtb,
	}, nil
}
