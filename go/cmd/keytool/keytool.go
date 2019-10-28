package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"gopkg.in/square/go-jose.v2"
)

var (
	action           = flag.String("action", "help", "The action to take.")
	headerJSON       = flag.String("header", `{"alg": "RS256","typ": "JWT"}`, "The JSON of the JWT header (for sign).")
	payloadJSON      = flag.String("payload", "", "The JSON of the JWT payload (for sign).")
	outputPrivateKey = flag.String("output_private_key", "", "The file name to write a key into (for createkey).")
	outputPublicKey  = flag.String("output_public_key", "", "The file name to write a key into (for createkey).")
	inputPrivateKey  = flag.String("input_private_key", "", "The file name to read a key from (for sign).")
	inputPublicKey   = flag.String("input_public_key", "", "The file name to read a key from (for verify).")
	token            = flag.String("token", "", "The input JWT token to verify.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		glog.Exit(err)
	}
}

func run() error {
	switch *action {
	case "sign":
		return signToken()
	case "verify":
		return verifyToken()
	case "createkey":
		return createKey()
	case "read":
		return readKey()
	case "help":
		fmt.Println(`Usage: keytool -action <action>. Available actions:
			* sign: sign a token
			* verify: verify a token
			* createkey: create an RSA key
			* read: read a key from Cloud Storage bucket

		Run keytool -help to see all command line flags.`)
		return nil
	}
	return fmt.Errorf("unknown action %q", *action)
}

func signToken() error {
	b, err := ioutil.ReadFile(*inputPrivateKey)
	if err != nil {
		return fmt.Errorf("error reading from %q: %s", *inputPrivateKey, err)
	}
	block, _ := pem.Decode(b)
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("error parsing key from %q: %s", *inputPrivateKey, err)
	}
	payload := make(map[string]string)
	err = json.Unmarshal([]byte(*payloadJSON), &payload)
	if err != nil {
		return fmt.Errorf("error parsing paylaod JSON %q: %s", *payloadJSON, err)
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error serializing payload JSON: %s", err)
	}
	opts := &jose.SignerOptions{}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: rsaKey}, opts.WithType("JWT"))
	if err != nil {
		return fmt.Errorf("error creating JWT signer: %s", err)
	}
	if err != nil {
		return err
	}
	object, err := signer.Sign(payloadBytes)
	if err != nil {
		return fmt.Errorf("error signing JWT token: %s", err)
	}
	//fmt.Println(base64.StdEncoding.EncodeToString(object))
	jwt, err := object.CompactSerialize()
	if err != nil {
		return fmt.Errorf("error serializing JWT signature: %s", err)
	}
	fmt.Println(jwt)
	return nil
}

func createKey() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	b := x509.MarshalPKCS1PrivateKey(privateKey)
	privTxt := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: b,
		},
	)
	fmt.Println(string(privTxt))
	if *outputPrivateKey != "" {
		if err := ioutil.WriteFile(*outputPrivateKey, privTxt, 0700); err != nil {
			return fmt.Errorf("error writing to %q: %s", *outputPrivateKey, err)
		}
	}
	pubTxt := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
		},
	)
	fmt.Println(string(pubTxt))
	if *outputPublicKey != "" {
		if err := ioutil.WriteFile(*outputPublicKey, pubTxt, 0700); err != nil {
			return fmt.Errorf("error writing to %q: %s", *outputPublicKey, err)
		}
	}
	return nil
}

func verifyToken() error {
	b, err := ioutil.ReadFile(*inputPublicKey)
	if err != nil {
		return fmt.Errorf("error reading from %q: %s", *inputPublicKey, err)
	}
	block, _ := pem.Decode(b)
	pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("error parsing key from %q: %s", *inputPrivateKey, err)
	}
	/*
		parts := strings.Split(*token, ".")
		if len(parts) != 3 {
			return fmt.Errorf("JWT token should have 3 dot-separated parts, got %d", len(parts))
		}
		hb, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			return fmt.Errorf("error decoding JWT header: %s", err)
		}
		fmt.Printf("header: %s\n", string(hb))
		header := make(map[string]string)
		err = json.Unmarshal(hb, &header)
		if err != nil {
			return fmt.Errorf("error parsing JWT header: %s", err)
		}
		pb, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("error decoding JWT payload: %s", err)
		}
	*/
	object, err := jose.ParseSigned(*token)
	if err != nil {
		return fmt.Errorf("error parsing JWT signature: %s", err)
	}
	pb, err := object.Verify(pubKey)
	if err != nil {
		return fmt.Errorf("error verifying JWT token: %s", err)
	}
	fmt.Printf("payload: %s\n", string(pb))
	payload := make(map[string]string)
	err = json.Unmarshal(pb, &payload)
	if err != nil {
		return fmt.Errorf("error parsing JWT payload: %s", err)
	}
	fmt.Println("OK")
	return nil
}

func readKey() error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating Cloud Storage client: %s", err)
	}
	parts := strings.SplitN(*inputPrivateKey, "/", 4)
	if len(parts) != 4 || parts[0] != "gs:" || parts[1] != "" {
		return fmt.Errorf("--input_private_key must have gs://bucket/keyfile format, got %q", *inputPrivateKey)
	}
	bucket := client.Bucket(parts[2])
	obj := bucket.Object(parts[3])
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("error reading from bucket object %q: %s", *inputPrivateKey, err)
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading from bucket object %q: %s", *inputPrivateKey, err)
	}
	block, _ := pem.Decode(b)
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("error parsing key from %q: %s", *inputPrivateKey, err)
	}
	fmt.Println(string(pem.EncodeToMemory(
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)},
	)))
	fmt.Println(string(pem.EncodeToMemory(
		&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&rsaKey.PublicKey)},
	)))
	return nil
}
