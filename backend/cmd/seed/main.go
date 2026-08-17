package main

import (
	_ "time/tzdata"

	"github.com/m-asjedh/event-management-platform/backend/internal/seed"
)

func main() {
	seed.Main()
}
