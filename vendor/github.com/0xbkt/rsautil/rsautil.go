package rsautil

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"strings"

	"golang.org/x/crypto/ssh"
)

// KeyPairFromFile calls KeyPair with the data it reads from filename.
func KeyPairFromFile(filename string) (privKey *rsa.PrivateKey, pubKey *rsa.PublicKey, sshPubKeyHuman string, err error) {
	privKeyRawPEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	return KeyPair(privKeyRawPEM)
}

// KeyPair returns a structured RSA keypair from the raw
// PEM key and an SSH public key corresponding to it.
func KeyPair(privKeyRawPEM []byte) (privKey *rsa.PrivateKey, pubKey *rsa.PublicKey, sshPubKeyHuman string, err error) {
	p, _ := pem.Decode(privKeyRawPEM)

	privKey, err = x509.ParsePKCS1PrivateKey(p.Bytes)
	if err != nil {
		return
	}

	sshPubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return
	}

	sshPubKeyHuman = strings.TrimSuffix(
		string(ssh.MarshalAuthorizedKey(sshPubKey)),
		"\n",
	)

	return
}
