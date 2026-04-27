package service

import (
	"errors"

	"github.com/lib/pq"
)

func isCheckViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23514"
	}
	return false
}
