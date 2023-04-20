package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Product holds the schema definition for the Product entity.
type Product struct {
	ent.Schema
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("key"),
		field.String("name"),
		field.Int("list_price"),
		field.Int("selling_price"),
		field.Int("credits"),
		field.Enum("status").Values("OnSale", "SoldOut", "Discontinued", "New"),
		field.String("description"),
		field.Time("created_at").
			Default(time.Now).
			StructTag(`json:"-"`),
	}
}

// Edges of the Product.
func (Product) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("orders", Order.Type).
			Ref("product"),
	}
}
