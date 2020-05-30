package repositories

import "errors"

var ErrRouteNotFound = errors.New("route not found")

var ErrRouteAlreadyAssigned = errors.New("route already assigned")
