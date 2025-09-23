package model

import "encoding/json"

type OrderAccrual struct {
	OrderID string      `json:"order"`
	Status  OrderStatus `json:"-"`
	Accrual float64     `json:"accrual"`
}

func (oa *OrderAccrual) UnmarshalJSON(data []byte) error {
	type Alias OrderAccrual
	aux := &struct {
		*Alias
		St string `json:"status"`
	}{
		Alias: (*Alias)(oa),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.St == "REGISTERED" {
		oa.Status = OrderStatusNew
	} else {
		oa.Status = OrderStatus(aux.St)
	}
	return nil
}
