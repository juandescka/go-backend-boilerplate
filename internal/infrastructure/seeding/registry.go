package seeding

import (
	"context"
	"database/sql"
	"maps"
	"slices"
)

type Dataset struct {
	Description string
	Run         func(context.Context, *sql.Tx) (created bool, err error)
}

var registry = map[string]Dataset{
	"dev-users":         {Description: "Create a passwordless development user", Run: devUsers},
	"demo-tenant":       {Description: "Create the demo tenant", Run: demoTenant},
	"demo-organization": {Description: "Create an organization in the demo tenant", Run: demoOrganization},
	"demo-memberships":  {Description: "Link the demo user to the demo tenant", Run: demoMemberships},
}

func Names() []string { return slices.Sorted(maps.Keys(registry)) }

func Lookup(name string) (Dataset, bool) {
	dataset, ok := registry[name]
	return dataset, ok
}
