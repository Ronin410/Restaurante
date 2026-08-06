package orders

import "testing"

func TestLineTotal(t *testing.T) {
	cases := []struct {
		name string
		item OrderItem
		want float64
	}{
		{
			name: "sin modificadores",
			item: OrderItem{Qty: 3, UnitPrice: 18},
			want: 54,
		},
		{
			name: "con modificadores que suman costo",
			item: OrderItem{
				Qty: 2, UnitPrice: 45,
				Modifiers: []ItemModifier{{Name: "Doble queso", PriceDelta: 8}, {Name: "Con todo", PriceDelta: 0}},
			},
			want: 106, // (45+8+0) * 2
		},
		{
			name: "cantidad fraccionaria",
			item: OrderItem{Qty: 0.5, UnitPrice: 100},
			want: 50,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lineTotal(tc.item)
			if got != tc.want {
				t.Fatalf("lineTotal() = %.2f, want %.2f", got, tc.want)
			}
		})
	}
}
