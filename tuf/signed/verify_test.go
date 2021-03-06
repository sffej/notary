package signed

import (
	"testing"
	"time"

	"github.com/docker/go/canonical/json"
	"github.com/docker/notary"
	"github.com/stretchr/testify/require"

	"github.com/docker/notary/tuf/data"
)

func TestRoleNoKeys(t *testing.T) {
	cs := NewEd25519()
	k, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{}, Threshold: 1}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k}, 1, nil))
	err = VerifySignatures(s, roleWithKeys)
	require.IsType(t, ErrRoleThreshold{}, err)
}

func TestNotEnoughSigs(t *testing.T) {
	cs := NewEd25519()
	k, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{k.ID(): k}, Threshold: 2}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k}, 1, nil))
	err = VerifySignatures(s, roleWithKeys)
	require.IsType(t, ErrRoleThreshold{}, err)
}

func TestNoSigs(t *testing.T) {
	cs := NewEd25519()
	k, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{k.ID(): k}, Threshold: 2}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.Equal(t, ErrNoSignatures, VerifySignatures(s, roleWithKeys))
}

func TestExactlyEnoughSigs(t *testing.T) {
	cs := NewEd25519()
	k, err := cs.Create(data.CanonicalRootRole, "", data.ED25519Key)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{
		Name: data.CanonicalRootRole, Keys: data.Keys{k.ID(): k}, Threshold: 1}

	meta := &data.SignedCommon{Type: data.TUFTypes[data.CanonicalRootRole], Version: 1,
		Expires: data.DefaultExpires(data.CanonicalRootRole)}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k}, 1, nil))
	require.Equal(t, 1, len(s.Signatures))

	require.NoError(t, VerifySignatures(s, roleWithKeys))
}

func TestMoreThanEnoughSigs(t *testing.T) {
	cs := NewEd25519()
	k1, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	k2, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{k1.ID(): k1, k2.ID(): k2}, Threshold: 1}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k1, k2}, 2, nil))
	require.Equal(t, 2, len(s.Signatures))

	err = VerifySignatures(s, roleWithKeys)
	require.NoError(t, err)
}

func TestValidSigWithIncorrectKeyID(t *testing.T) {
	cs := NewEd25519()
	k1, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{"invalidIDA": k1}, Threshold: 1}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k1}, 1, nil))
	require.Equal(t, 1, len(s.Signatures))
	s.Signatures[0].KeyID = "invalidIDA"
	err = VerifySignatures(s, roleWithKeys)
	require.Error(t, err)
	require.IsType(t, ErrInvalidKeyID{}, err)
}

func TestDuplicateSigs(t *testing.T) {
	cs := NewEd25519()
	k, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{k.ID(): k}, Threshold: 2}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k}, 1, nil))
	s.Signatures = append(s.Signatures, s.Signatures[0])
	err = VerifySignatures(s, roleWithKeys)
	require.IsType(t, ErrRoleThreshold{}, err)
}

func TestUnknownKeyBelowThreshold(t *testing.T) {
	cs := NewEd25519()
	k, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	unknown, err := cs.Create("root", "", data.ED25519Key)
	require.NoError(t, err)
	roleWithKeys := data.BaseRole{Name: "root", Keys: data.Keys{k.ID(): k}, Threshold: 2}

	meta := &data.SignedCommon{Type: "Root", Version: 1, Expires: data.DefaultExpires("root")}

	b, err := json.MarshalCanonical(meta)
	require.NoError(t, err)
	s := &data.Signed{Signed: (*json.RawMessage)(&b)}
	require.NoError(t, Sign(cs, s, []data.PublicKey{k, unknown}, 2, nil))
	s.Signatures = append(s.Signatures)
	err = VerifySignatures(s, roleWithKeys)
	require.IsType(t, ErrRoleThreshold{}, err)
}

func TestVerifyVersion(t *testing.T) {
	tufType := data.TUFTypes[data.CanonicalRootRole]
	meta := data.SignedCommon{Type: tufType, Version: 1, Expires: data.DefaultExpires(data.CanonicalRootRole)}
	require.Equal(t, ErrLowVersion{Actual: 1, Current: 2}, VerifyVersion(&meta, 2))
	require.NoError(t, VerifyVersion(&meta, 1))
}

func TestVerifyExpiry(t *testing.T) {
	tufType := data.TUFTypes[data.CanonicalRootRole]
	notExpired := data.DefaultExpires(data.CanonicalRootRole)
	expired := time.Now().Add(-1 * notary.Year)

	require.NoError(t, VerifyExpiry(
		&data.SignedCommon{Type: tufType, Version: 1, Expires: notExpired}, data.CanonicalRootRole))
	err := VerifyExpiry(
		&data.SignedCommon{Type: tufType, Version: 1, Expires: expired}, data.CanonicalRootRole)
	require.Error(t, err)
	require.IsType(t, ErrExpired{}, err)
}
