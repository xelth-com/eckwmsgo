package sync

// ChecksumCalculator stubbed for compilation during refactor
type ChecksumCalculator struct {
	instanceID string
}

// NewChecksumCalculator creates a new checksum calculator
func NewChecksumCalculator(instanceID string) *ChecksumCalculator {
	return &ChecksumCalculator{instanceID: instanceID}
}

// TODO: Re-implement checksums for new Odoo models (ProductProduct, StockLocation, StockLot, StockQuantPackage, StockQuant)
