package zerrors_test

import (
	"errors"
	"testing"

	"github.com/DeluxeOwl/zerrors"
	"github.com/stretchr/testify/require"
)

func Test_ErrorCreation(t *testing.T) {
	type domainErr string

	const (
		domainErrNotFound   domainErr = "not_found"
		domainErrBadRequest domainErr = "bad_request"
	)

	err := zerrors.
		New(domainErrNotFound).
		With("user_id", 123).
		With("trace", "1234").
		Errorf("message")

	type dbErr string
	const (
		dbErrZeroRows dbErr = "zero_rows"
	)

	errDB := zerrors.
		New(dbErrZeroRows).
		With("req_id", 10).
		Errorf("db returned no rows")

	err = err.WithError(errDB)

	code, asCB := zerrors.As(err, func(zerr *zerrors.Error[domainErr]) domainErr {
		require.Equal(t, domainErrNotFound, zerr.Code())
		userID, ok := zerr.Get("user_id")
		require.True(t, ok)
		require.Equal(t, 123, userID)
		return zerr.Code()
	})
	require.Equal(t, domainErrNotFound, *code)
	require.True(t, asCB)

	var derr *zerrors.Error[domainErr]
	if errors.As(err, &derr) {
		require.Equal(t, domainErrNotFound, derr.Code())
		userID, ok := derr.Get("user_id")
		require.True(t, ok)
		require.Equal(t, 123, userID)
	}

	var dberr *zerrors.Error[dbErr]
	if errors.As(err, &dberr) {
		require.Equal(t, dbErrZeroRows, dberr.Code())
		reqID, ok := dberr.Get("req_id")
		require.True(t, ok)
		require.Equal(t, 10, reqID)

		require.True(t, zerrors.HasCode(err, dbErrZeroRows))
		require.True(t, zerrors.HasCode(err, domainErrNotFound))
	}
}

func Test_Tags(t *testing.T) {
	type anotherErr string

	const (
		anotherErrUnauthorized anotherErr = "unauthorized"
	)

	err0 := zerrors.
		New(anotherErrUnauthorized).
		Tags("permission")

	type domainErr string

	const (
		domainErrNotFound domainErr = "not_found"
	)

	err := zerrors.
		New(domainErrNotFound).
		Tags("iam", "authz").WithError(err0)

	var derr *zerrors.Error[domainErr]
	if errors.As(err, &derr) {
		require.True(t, derr.HasTags("iam"))
		require.True(t, derr.HasTags("authz"))
		require.True(t, derr.HasTags("permission"))
		require.ElementsMatch(t, []string{"permission", "iam", "authz"}, derr.GetTags())
	}
}
