package service

import (
    "encoding/json"
    "github.com/pearsonspecter/rules-resolution/internal/domain"
    "github.com/pearsonspecter/rules-resolution/internal/repository"
)

type ResolutionService struct {
    defaults     map[string]map[domain.TraitKey]json.RawMessage
    overrideRepo *repository.OverrideRepository
}

func NewResolutionService(
    defaults map[string]map[domain.TraitKey]json.RawMessage,
    overrideRepo *repository.OverrideRepository,
) *ResolutionService {
    return &ResolutionService{defaults: defaults, overrideRepo: overrideRepo}
}

func (s *ResolutionService) Resolve(ctx domain.CaseContext) domain.ResolutionResponse {
    return domain.ResolutionResponse{}
}
