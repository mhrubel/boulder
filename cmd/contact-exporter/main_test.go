package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/core"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/sa"
	"github.com/letsencrypt/boulder/test"
	"github.com/letsencrypt/boulder/test/vars"
	"github.com/square/go-jose"
)

func TestFindContacts(t *testing.T) {
	testCtx := setup(t)
	defer testCtx.cleanUp()

	ctx := context.Background()

	const emailARaw = "test@example.com"
	const emailBRaw = "example@example.com"
	const emailCRaw = "test-example@example.com"
	emailA, _ := core.ParseAcmeURL("mailto:" + emailARaw)
	emailB, _ := core.ParseAcmeURL("mailto:" + emailBRaw)
	emailC, _ := core.ParseAcmeURL("mailto:" + emailCRaw)
	tel, _ := core.ParseAcmeURL("tel:666-666-7777")

	jsonKeyA := []byte(`{
  "kty":"RSA",
  "n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
  "e":"AQAB"
}`)
	jsonKeyB := []byte(`{
  "kty":"RSA",
  "n":"z8bp-jPtHt4lKBqepeKF28g_QAEOuEsCIou6sZ9ndsQsEjxEOQxQ0xNOQezsKa63eogw8YS3vzjUcPP5BJuVzfPfGd5NVUdT-vSSwxk3wvk_jtNqhrpcoG0elRPQfMVsQWmxCAXCVRz3xbcFI8GTe-syynG3l-g1IzYIIZVNI6jdljCZML1HOMTTW4f7uJJ8mM-08oQCeHbr5ejK7O2yMSSYxW03zY-Tj1iVEebROeMv6IEEJNFSS4yM-hLpNAqVuQxFGetwtwjDMC1Drs1dTWrPuUAAjKGrP151z1_dE74M5evpAhZUmpKv1hY-x85DC6N0hFPgowsanmTNNiV75w",
  "e":"AAEAAQ"
}`)
	jsonKeyC := []byte(`{
  "kty":"RSA",
  "n":"rFH5kUBZrlPj73epjJjyCxzVzZuV--JjKgapoqm9pOuOt20BUTdHqVfC2oDclqM7HFhkkX9OSJMTHgZ7WaVqZv9u1X2yjdx9oVmMLuspX7EytW_ZKDZSzL-sCOFCuQAuYKkLbsdcA3eHBK_lwc4zwdeHFMKIulNvLqckkqYB9s8GpgNXBDIQ8GjR5HuJke_WUNjYHSd8jY1LU9swKWsLQe2YoQUz_ekQvBvBCoaFEtrtRaSJKNLIVDObXFr2TLIiFiM0Em90kK01-eQ7ZiruZTKomll64bRFPoNo4_uwubddg3xTqur2vdF3NyhTrYdvAgTem4uC0PFjEQ1bK_djBQ",
  "e":"AQAB"
}`)
	jsonKeyD := []byte(`{
  "kty":"RSA",
  "n":"rFH5kUBZrlPj73epjJjyCxzVzZuV--JjKgapoqm9pOuOt20BUTdHqVfC2oDclqM7HFhkkX9OSJMTHgZ7WaVqZv9u1X2yjdx9oVmMLuspX7EytW_ZKDZSzL-FCOFCuQAuYKkLbsdcA3eHBK_lwc4zwdeHFMKIulNvLqckkqYB9s8GpgNXBDIQ8GjR5HuJke_WUNjYHSd8jY1LU9swKWsLQe2YoQUz_ekQvBvBCoaFEtrtRaSJKNLIVDObXFr2TLIiFiM0Em90kK01-eQ7ZiruZTKomll64bRFPoNo4_uwubddg3xTqur2vdF3NyhTrYdvAgTem4uC0PFjEQ1bK_djBQ",
  "e":"AQAB"
}`)

	var keyA jose.JsonWebKey
	var keyB jose.JsonWebKey
	var keyC jose.JsonWebKey
	var keyD jose.JsonWebKey
	err := json.Unmarshal(jsonKeyA, &keyA)
	test.AssertNotError(t, err, "Failed to unmarshal public JWK")
	err = json.Unmarshal(jsonKeyB, &keyB)
	test.AssertNotError(t, err, "Failed to unmarshal public JWK")
	err = json.Unmarshal(jsonKeyC, &keyC)
	test.AssertNotError(t, err, "Failed to unmarshal public JWK")
	err = json.Unmarshal(jsonKeyD, &keyD)
	test.AssertNotError(t, err, "Failed to unmarshal public JWK")

	// Add some test registrations
	regA := core.Registration{
		ID: 1,
		Contact: &[]*core.AcmeURL{
			emailA,
		},
		Key:       keyA,
		InitialIP: net.ParseIP("2.3.2.3"),
	}
	regB := core.Registration{
		ID: 2,
		Contact: &[]*core.AcmeURL{
			emailB,
		},
		Key:       keyB,
		InitialIP: net.ParseIP("2.3.2.3"),
	}
	regC := core.Registration{
		ID: 3,
		Contact: &[]*core.AcmeURL{
			emailC,
		},
		Key:       keyC,
		InitialIP: net.ParseIP("210.3.2.3"),
	}
	regD := core.Registration{
		ID: 4,
		Contact: &[]*core.AcmeURL{
			tel,
		},
		Key:       keyD,
		InitialIP: net.ParseIP("2.3.2.3"),
	}
	regA, err = testCtx.ssa.NewRegistration(ctx, regA)
	if err != nil {
		t.Fatalf("Couldn't store regA: %s", err)
	}
	regB, err = testCtx.ssa.NewRegistration(ctx, regB)
	if err != nil {
		t.Fatalf("Couldn't store regB: %s", err)
	}
	regC, err = testCtx.ssa.NewRegistration(ctx, regC)
	if err != nil {
		t.Fatalf("Couldn't store regC: %s", err)
	}
	regD, err = testCtx.ssa.NewRegistration(ctx, regD)
	if err != nil {
		t.Fatalf("Couldn't store regD: %s", err)
	}

	// Run findContacts - since no certificates have been added corresponding to
	// the above registrations, no contacts should be found.
	contacts, err := testCtx.c.findContacts()
	test.AssertNotError(t, err, "findContacts() produced error")
	test.AssertEquals(t, len(contacts), 0)

	var serial1 = big.NewInt(1336)
	var serial1String = core.SerialToString(serial1)
	var serial2 = big.NewInt(1337)
	var serial2String = core.SerialToString(serial2)
	var serial3 = big.NewInt(1338)
	var serial3String = core.SerialToString(serial3)
	var serial4 = big.NewInt(1339)
	var serial4String = core.SerialToString(serial4)
	var n = bigIntFromB64("n4EPtAOCc9AlkeQHPzHStgAbgs7bTZLwUBZdR8_KuKPEHLd4rHVTeT-O-XV2jRojdNhxJWTDvNd7nqQ0VEiZQHz_AJmSCpMaJMRBSFKrKb2wqVwGU_NsYOYL-QtiWN2lbzcEe6XC0dApr5ydQLrHqkHHig3RBordaZ6Aj-oBHqFEHYpPe7Tpe-OfVfHd1E6cS6M1FZcD1NNLYD5lFHpPI9bTwJlsde3uhGqC0ZCuEHg8lhzwOHrtIQbS0FVbb9k3-tVTU4fg_3L_vniUFAKwuCLqKnS2BYwdq_mzSnbLY7h_qixoR7jig3__kRhuaxwUkRz5iaiQkqgc5gHdrNP5zw==")
	var e = intFromB64("AQAB")
	var d = bigIntFromB64("bWUC9B-EFRIo8kpGfh0ZuyGPvMNKvYWNtB_ikiH9k20eT-O1q_I78eiZkpXxXQ0UTEs2LsNRS-8uJbvQ-A1irkwMSMkK1J3XTGgdrhCku9gRldY7sNA_AKZGh-Q661_42rINLRCe8W-nZ34ui_qOfkLnK9QWDDqpaIsA-bMwWWSDFu2MUBYwkHTMEzLYGqOe04noqeq1hExBTHBOBdkMXiuFhUq1BU6l-DqEiWxqg82sXt2h-LMnT3046AOYJoRioz75tSUQfGCshWTBnP5uDjd18kKhyv07lhfSJdrPdM5Plyl21hsFf4L_mHCuoFau7gdsPfHPxxjVOcOpBrQzwQ==")
	var p = bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc=")
	var q = bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc=")

	var testKey = rsa.PrivateKey{
		PublicKey: rsa.PublicKey{N: n, E: e},
		D:         d,
		Primes:    []*big.Int{p, q},
	}

	fc := newFakeClock(t)

	// Add one cert for RegA that expires in 30 days
	rawCertA := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "happy A",
		},
		NotAfter:     fc.Now().Add(30 * 24 * time.Hour),
		DNSNames:     []string{"example-a.com"},
		SerialNumber: serial1,
	}
	certDerA, _ := x509.CreateCertificate(rand.Reader, &rawCertA, &rawCertA, &testKey.PublicKey, &testKey)
	certA := &core.Certificate{
		RegistrationID: regA.ID,
		Serial:         serial1String,
		Expires:        rawCertA.NotAfter,
		DER:            certDerA,
	}
	err = testCtx.c.dbMap.Insert(certA)
	test.AssertNotError(t, err, "Couldn't add certA")

	// Add one cert for RegB that already expired 30 days ago
	rawCertB := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "happy B",
		},
		NotAfter:     fc.Now().Add(-30 * 24 * time.Hour),
		DNSNames:     []string{"example-b.com"},
		SerialNumber: serial2,
	}
	certDerB, _ := x509.CreateCertificate(rand.Reader, &rawCertB, &rawCertB, &testKey.PublicKey, &testKey)
	certB := &core.Certificate{
		RegistrationID: regB.ID,
		Serial:         serial2String,
		Expires:        rawCertB.NotAfter,
		DER:            certDerB,
	}
	err = testCtx.c.dbMap.Insert(certB)
	test.AssertNotError(t, err, "Couldn't add certB")

	// Add one cert for RegC that expires in 30 days
	rawCertC := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "happy C",
		},
		NotAfter:     fc.Now().Add(30 * 24 * time.Hour),
		DNSNames:     []string{"example-c.com"},
		SerialNumber: serial3,
	}
	certDerC, _ := x509.CreateCertificate(rand.Reader, &rawCertC, &rawCertC, &testKey.PublicKey, &testKey)
	certC := &core.Certificate{
		RegistrationID: regC.ID,
		Serial:         serial3String,
		Expires:        rawCertC.NotAfter,
		DER:            certDerC,
	}
	err = testCtx.c.dbMap.Insert(certC)
	test.AssertNotError(t, err, "Couldn't add certC")

	// Add one cert for RegD that expires in 30 days
	rawCertD := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "happy D",
		},
		NotAfter:     fc.Now().Add(30 * 24 * time.Hour),
		DNSNames:     []string{"example-d.com"},
		SerialNumber: serial4,
	}
	certDerD, _ := x509.CreateCertificate(rand.Reader, &rawCertD, &rawCertD, &testKey.PublicKey, &testKey)
	certD := &core.Certificate{
		RegistrationID: regD.ID,
		Serial:         serial4String,
		Expires:        rawCertD.NotAfter,
		DER:            certDerD,
	}
	err = testCtx.c.dbMap.Insert(certD)
	test.AssertNotError(t, err, "Couldn't add certD")

	// Run findContacts - since there are two registrations with unexpired certs
	// we should get exactly two contacts back for RegA and RegC. RegB should
	// *not* be present since their certificate has already expired. Similarly,
	// RegD should *not* be present since its only contact is a "tel:" prefixed
	// ACMEUrl.
	contacts, err = testCtx.c.findContacts()
	test.AssertNotError(t, err, "findContacts() produced error")
	test.AssertEquals(t, len(contacts), 2)
	test.AssertEquals(t, contacts[0], emailARaw)
	test.AssertEquals(t, contacts[1], emailCRaw)
}

func TestWriteContacts(t *testing.T) {
	expectedOutput := `
example@example.com
test-example@example.com
test-test-test@example.com
`
	contacts := strings.Split(expectedOutput, "\n")

	dir := os.TempDir()
	f, err := ioutil.TempFile(dir, "contacts_test")
	test.AssertNotError(t, err, "ioutil.TempFile produced an error")

	// Writing the contacts with no outFile should print to stdout
	err = writeContacts(contacts, "")
	test.AssertNotError(t, err, "writeContacts() with no outfile produced error")

	// Writing the contacts to an outFile should produce the correct results
	err = writeContacts(contacts, f.Name())
	test.AssertNotError(t, err, fmt.Sprintf("writeContacts() produced an error writing to %s", f.Name()))

	contents, err := ioutil.ReadFile(f.Name())
	test.AssertNotError(t, err, fmt.Sprintf("ioutil.ReadFile produced an error reading from %s", f.Name()))

	test.AssertEquals(t, string(contents), expectedOutput)
}

type testCtx struct {
	c       contactExporter
	ssa     core.StorageAdder
	cleanUp func()
}

func setup(t *testing.T) testCtx {
	log := blog.UseMock()

	// Using DBConnSAFullPerms to be able to insert registrations and certificates
	dbMap, err := sa.NewDbMap(vars.DBConnSAFullPerms, 0)
	if err != nil {
		t.Fatalf("Couldn't connect the database: %s", err)
	}
	cleanUp := test.ResetSATestDatabase(t)

	fc := newFakeClock(t)
	ssa, err := sa.NewSQLStorageAuthority(dbMap, fc, log)
	if err != nil {
		t.Fatalf("unable to create SQLStorageAuthority: %s", err)
	}

	return testCtx{
		c: contactExporter{
			dbMap: dbMap,
			log:   log,
			clk:   fc,
		},
		ssa:     ssa,
		cleanUp: cleanUp,
	}
}

func bigIntFromB64(b64 string) *big.Int {
	bytes, _ := base64.URLEncoding.DecodeString(b64)
	x := big.NewInt(0)
	x.SetBytes(bytes)
	return x
}

func intFromB64(b64 string) int {
	return int(bigIntFromB64(b64).Int64())
}

func newFakeClock(t *testing.T) clock.FakeClock {
	const fakeTimeFormat = "2006-01-02T15:04:05.999999999Z"
	ft, err := time.Parse(fakeTimeFormat, fakeTimeFormat)
	if err != nil {
		t.Fatal(err)
	}
	fc := clock.NewFake()
	fc.Set(ft.UTC())
	return fc
}
