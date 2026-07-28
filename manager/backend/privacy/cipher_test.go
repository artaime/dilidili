package privacy

import (
	"os"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := Encrypt(dek, "k1", "你好世界")
	if err != nil {
		t.Fatal(err)
	}
	if !IsCiphertext(ct) {
		t.Fatalf("expected ciphertext, got %q", ct)
	}
	pt, err := Decrypt(dek, ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "你好世界" {
		t.Fatalf("got %q", pt)
	}
}

func TestDecryptPlaintextPassthrough(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	pt, err := Decrypt(dek, "明文留言")
	if err != nil {
		t.Fatal(err)
	}
	if pt != "明文留言" {
		t.Fatalf("got %q", pt)
	}
}

func TestDecryptTamperedFails(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := Encrypt(dek, "k1", "secret")
	if err != nil {
		t.Fatal(err)
	}
	// flip last char of base64
	tampered := ct[:len(ct)-1] + "A"
	if _, err := Decrypt(dek, tampered); err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestWrapUnwrapDEK(t *testing.T) {
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(i)
	}
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := WrapDEK(kek, dek)
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnwrapDEK(kek, wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(dek) {
		t.Fatal("DEK mismatch after wrap/unwrap")
	}
}

func TestEncryptDecryptBytes(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte{0x00, 0x01, 0xff, 0x7f}
	ct, err := EncryptBytes(dek, "k1", plain)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecryptBytes(dek, []byte(ct))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("bytes mismatch")
	}
	// plaintext file passthrough
	raw := []byte("RIFF....WAVE")
	out2, err := DecryptBytes(dek, raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(out2) != string(raw) {
		t.Fatal("passthrough failed")
	}
}

func TestEmptyEncrypt(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := Encrypt(dek, "k1", "")
	if err != nil || ct != "" {
		t.Fatalf("empty encrypt: %q %v", ct, err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
