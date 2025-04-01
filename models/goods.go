package models

type Goods struct {
	ID          string  `json:"good_id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	SellerID    string  `json:"merchant_id"`
	Seller      string  `json:"merchant_name"`
	Picture     string  `json:"picture"`
	Description string  `json:"full_desc"`
	Tag         string  `json:"tag"`
}
