package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("ip"),
		field.String("token").Unique(),
		field.Int("credits"),
		field.Int("free_credits"),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Optional().
			Default(time.Now).
			UpdateDefault(time.Now).
			StructTag(`json:"-"`),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("orders", Order.Type).
			Ref("user"),
	}
}
