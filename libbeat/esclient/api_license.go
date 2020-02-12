package client

import (
	"encoding/json"
)

// License
type License struct {
	Status string
	Type   string
}

func (c *Client) GetLicense() (*License, error) {
	var rs struct {
		License License `json:"license"`
	}

	switch {
	case c.e6 != nil:
		r, err := c.e6.XPack.LicenseGet()
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		if r.IsError() {
			return nil, errorFromBody(r.Body)
		}

		d := json.NewDecoder(r.Body)
		if err := d.Decode(&rs); err != nil {
			return nil, err
		}

	case c.e7 != nil:
		r, err := c.e7.License.Get()
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		if r.IsError() {
			return nil, errorFromBody(r.Body)
		}

		d := json.NewDecoder(r.Body)
		if err := d.Decode(&rs); err != nil {
			return nil, err
		}

	case c.e8 != nil:
		r, err := c.e8.License.Get()
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		if r.IsError() {
			return nil, errorFromBody(r.Body)
		}

		d := json.NewDecoder(r.Body)
		if err := d.Decode(&rs); err != nil {
			return nil, err
		}

	default:
		return nil, ErrUnsupportedVersion
	}

	return &rs.License, nil
}
