package models

type PlayerDocument struct {
	Name      string
	Type      string
	Inventory InventoryDocument
}

type InventoryDocument struct {
	MaxCapacity     int64
	CurrentCapacity int64
	Items           []ItemDocument
}

type ItemDocument struct {
	Quantity int64
	Weight   int64
	SellCost int64
}
