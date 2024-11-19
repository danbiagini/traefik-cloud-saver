package common

import (
	"testing"
)

func TestJWTSigner(t *testing.T) {
	// Create a test private key in PEM format
	privateKeyPEM := `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDT5zFk8EWCOFkG
TMWdjq24qVPAoSCgiraieOsPZnEn2tFrxSTlwqd4PZ9KTE+TGgd0UxJ6C3dMjDTp
HREAP3nwIl2lcyLtiZX2L5ugJY7A+wMUBDsrvKzjG6eIvy8JuDSM44Z5E69EB4mB
WJ7pM0Ql+cDFjhHrhcL6yVVA7yq9YGZzuPnddkX/nkYhc2Deay3mwtlLJ9pB6B3r
O1u4RoCxxOUmXN6xxnbgZPhc91rtjFnnUZy8l8mr+ePam/INrPohzS/MHrjxOqFr
HIhUT8tkjkUFXionHR2bUSL3rjibTtG2gyTxsF4Jgge2GaJpw3fZEu6Hd3Q8/Hpf
ZvWBCmjhAgMBAAECggEABARAmy9zvddoFpa7czRaZiLth7v2JJzkf9lnaTwRnoYH
DLIotCM3re+LFqsyIfFvHT0a303a1cxdj2Kh6R2K5+qa2FFBovuF6Gv4GPXFSjKn
2QgIgBh8NXTXfN/U2iwP8PC6Io7lNlanPFir8HMskkS06vm5rLt1GfBZiZjO+FZz
3Xp2/RQ1aauKeHr2Z10N97ZETvwTokujrYz6my349dgPDzMxXcuG/FGhLJF945vk
RP2zaa3jPc9P47J5jZS4sqTSWC7oK56qjcdduL+ERpYHdLSC9TjdqGI8xu1dJqoF
YZJJ64CCPQ+8525sUefHeRpqhlNBaJr2E0/3e2E/jQKBgQDvOfR8doMoA2J6HaJw
zcv83BGM7hEuncorj9QUgWZmuxwxw3JSZy7RJOZfwzfX7ZLbkrD/pLKwBLe6geuL
tQza43Yjk+JqS678Gy/enADt10iHdrXdDU3z1f8llt7b58tX/eLVGl6YwYfD/cL0
0IE7+3756FmhtvXbDGU72OgMxQKBgQDiwsrvxCjAJFNCw0au+vDVSELwgRKgpGvZ
fErBb/FU41XTbf4PaSpRHiFmH03D/yTYqxVF/6ue8eYohS2kuYdIpDCd3TNSTI24
5qKhJOnUuImVOIYPBfWEs+B1ly97MegilTKGmQwRGnB1CnHUHnevPeP9iUQayE6B
l7mWwhqlbQKBgQCBnAG1CTSIEkVhafrfaPBy//xWQYl3my+0qEk8DtufHxLodz7S
HGtGDtrt2UPBLksZwYE6EE5rhTLRzqACYkYjtYcFQZMzCew1VLl7v0PVmIUIN63S
pOmuCSwifnoh5JTMCJbD5HSKCJh4/FyK7QiHqfuihFtDfW/4jN+wLBWVDQKBgQCN
U1QBXOL85WbS7DuIYLNqae/2TqtaXT8uO82ng2oIOutJq3q1BhkulzW/nPDtn33K
X84RY0gF9sM4K9CHom2TM2ltaehLeZS2UV+4SPZG8oAk9SZwBInBHA5fm0snX7JK
o2vrAUXI/w5pk4nf5uE24b7PTBabDo4HLJWpRO4wfQKBgDkbPMyIiyNrKbXRMZgM
Z2mkDyz8J4m2u91R5YwAldn1+97Mi0nX988JV6vbyDUnGmqfcwoQrYEKzgJgCTk7
NWZoexjHRye47uwcTvkYf4+ODkZo1cxgem553/sFveLYwLpse1F/FrxrZ+qUwJMT
G80WFPQ8buzddXhgsyQRDLjm
-----END PRIVATE KEY-----`

	signer, err := NewJWTSigner(privateKeyPEM)
	if err != nil {
		t.Fatalf("NewJWTSigner() error = %v", err)
	}

	// Test signing claims
	claims := map[string]interface{}{
		"sub":  "1234567890",
		"name": "Test User",
	}

	token, err := signer.SignClaims(claims)
	if err != nil {
		t.Fatalf("SignClaims() error = %v", err)
	}

	if token == "" {
		t.Error("SignClaims() returned empty token")
	}
}
