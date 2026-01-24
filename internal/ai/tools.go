package ai

import (
	"fmt"
	"log"

	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// ToolService provides inventory tools for AI
type ToolService struct {
	db *database.DB
}

func NewToolService(db *database.DB) *ToolService {
	return &ToolService{db: db}
}

// LinkCode links an external code to an internal ID
func (s *ToolService) LinkCode(internalID, externalCode, linkType, context string) error {
	log.Printf("🛠️ Tool Exec: LinkCode %s -> %s (%s)", externalCode, internalID, linkType)

	var exists models.ProductAlias
	if err := s.db.Where("external_code = ? AND internal_id = ?", externalCode, internalID).First(&exists).Error; err == nil {
		return fmt.Errorf("alias already exists")
	}

	alias := models.ProductAlias{
		ExternalCode:    externalCode,
		InternalID:      internalID,
		Type:            linkType,
		IsVerified:      true,
		ConfidenceScore: 100,
		CreatedContext:  context,
	}

	return s.db.Create(&alias).Error
}

// SearchInventory looks for existing aliases
func (s *ToolService) SearchInventory(query string) (*models.ProductAlias, bool) {
	var alias models.ProductAlias
	if err := s.db.Where("external_code = ?", query).First(&alias).Error; err == nil {
		return &alias, true
	}
	return nil, false
}
