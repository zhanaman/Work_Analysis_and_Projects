package domain

import "time"

// Partner represents an HPE channel partner with all associated data.
type Partner struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PartnerID string `json:"partner_id"` // HPE Partner ID

	// Tier info
	Tier string `json:"tier"` // Silver / Gold / Platinum

	// Location
	Country string `json:"country"`
	City    string `json:"city"`

	// Certifications
	ComputeCert    string `json:"compute_cert"`
	NetworkingCert string `json:"networking_cert"`
	HybridCloud    string `json:"hybrid_cloud_cert"`
	StorageCert    string `json:"storage_cert"`

	// Volumes
	RevenueYTD float64 `json:"revenue_ytd"`
	Target     float64 `json:"target"`

	// Contact
	ContactName  string `json:"contact_name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`

	// Metadata
	RawData    map[string]interface{} `json:"raw_data"`
	ImportedAt time.Time              `json:"imported_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}
