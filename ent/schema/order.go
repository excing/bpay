package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Order holds the schema definition for the Order entity.
type Order struct {
	ent.Schema
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("ip").Optional(),
		field.Int("fee"),
		field.Int("credits"),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Optional().
			Default(time.Now).
			UpdateDefault(time.Now).
			StructTag(`json:"-"`),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Required().
			Unique(),
	}
}
