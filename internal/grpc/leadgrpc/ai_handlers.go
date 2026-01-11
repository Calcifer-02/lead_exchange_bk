package leadgrpc

import (
	"context"
	"encoding/json"
	"fmt"

	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetClarificationQuestions — получить уточняющие вопросы для "короткого" лида.
func (s *serverAPI) GetClarificationQuestions(ctx context.Context, in *pb.GetClarificationQuestionsRequest) (*pb.GetClarificationQuestionsResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Проверяем, доступен ли агент уточнения
	if s.clarificationAgent == nil {
		return nil, status.Error(codes.Unavailable, "clarification service is not available")
	}

	leadID, err := uuid.Parse(in.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id format")
	}

	// Получаем лид
	lead, err := s.leadService.GetLead(ctx, leadID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("lead not found: %v", err))
	}

	// Анализируем и генерируем вопросы
	result, err := s.clarificationAgent.AnalyzeAndGenerateQuestions(ctx, lead)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to analyze lead: %v", err))
	}

	// Конвертируем в protobuf ответ
	resp := &pb.GetClarificationQuestionsResponse{
		NeedsClarification: result.NeedsClarification,
		Priority:           result.Priority,
		MissingFields:      result.MissingFields,
		LeadQualityScore:   result.LeadQualityScore,
	}

	for _, q := range result.Questions {
		resp.Questions = append(resp.Questions, &pb.ClarificationQuestion{
			Field:            q.Field,
			Question:         q.Question,
			QuestionType:     q.QuestionType,
			SuggestedOptions: q.SuggestedOptions,
			Importance:       q.Importance,
		})
	}

	return resp, nil
}

// ApplyClarificationAnswers — применить ответы на уточняющие вопросы.
func (s *serverAPI) ApplyClarificationAnswers(ctx context.Context, in *pb.ApplyClarificationAnswersRequest) (*pb.ApplyClarificationAnswersResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	leadID, err := uuid.Parse(in.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id format")
	}

	// Получаем текущий лид
	lead, err := s.leadService.GetLead(ctx, leadID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("lead not found: %v", err))
	}

	// Парсим текущий requirement
	var requirement map[string]interface{}
	if len(lead.Requirement) > 0 {
		if err := json.Unmarshal(lead.Requirement, &requirement); err != nil {
			requirement = make(map[string]interface{})
		}
	} else {
		requirement = make(map[string]interface{})
	}

	// Применяем ответы
	for _, answer := range in.Answers {
		field := answer.GetField()
		value := answer.GetValue()

		// Обрабатываем специальные поля
		switch field {
		case "price", "min_price", "max_price":
			// Пытаемся парсить как число
			var numValue int64
			if _, err := fmt.Sscanf(value, "%d", &numValue); err == nil {
				requirement[field] = numValue
			} else {
				requirement[field] = value
			}
		case "rooms", "min_rooms", "max_rooms":
			var numValue int32
			if _, err := fmt.Sscanf(value, "%d", &numValue); err == nil {
				requirement[field] = numValue
			} else {
				requirement[field] = value
			}
		case "area", "min_area", "max_area":
			var numValue float64
			if _, err := fmt.Sscanf(value, "%f", &numValue); err == nil {
				requirement[field] = numValue
			} else {
				requirement[field] = value
			}
		default:
			requirement[field] = value
		}
	}

	// Сериализуем обновлённый requirement
	newRequirement, err := json.Marshal(requirement)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to serialize requirement: %v", err))
	}

	// TODO: Обновить лид в базе данных
	// Пока возвращаем успешный результат с новым requirement
	// В будущем нужно добавить leadService.UpdateLeadRequirement

	return &pb.ApplyClarificationAnswersResponse{
		Success:        true,
		NewRequirement: newRequirement,
		Message:        fmt.Sprintf("Applied %d clarification answers", len(in.Answers)),
	}, nil
}

// AnalyzeLeadIntent — анализ намерений лида для определения оптимальных весов матчинга.
func (s *serverAPI) AnalyzeLeadIntent(ctx context.Context, in *pb.AnalyzeLeadIntentRequest) (*pb.AnalyzeLeadIntentResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Проверяем, доступен ли анализатор весов
	if s.weightsAnalyzer == nil {
		return nil, status.Error(codes.Unavailable, "weights analysis service is not available")
	}

	leadID, err := uuid.Parse(in.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id format")
	}

	// Получаем лид
	lead, err := s.leadService.GetLead(ctx, leadID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("lead not found: %v", err))
	}

	// Анализируем лид
	result, err := s.weightsAnalyzer.AnalyzeLead(ctx, lead)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to analyze lead: %v", err))
	}

	// Формируем ответ
	resp := &pb.AnalyzeLeadIntentResponse{
		RecommendedWeights: &pb.MatchWeights{
			Price:    result.Weights.Price,
			District: result.Weights.District,
			Rooms:    result.Weights.Rooms,
			Area:     result.Weights.Area,
			Semantic: result.Weights.Semantic,
		},
		LeadType:    result.LeadType,
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		UsedLlm:     result.UsedLLM,
	}

	// Добавляем извлечённые критерии, если есть
	if result.Criteria != nil {
		resp.ExtractedCriteria = &pb.ExtractedCriteria{
			TargetPrice:         result.Criteria.TargetPrice,
			TargetDistrict:      result.Criteria.TargetDistrict,
			TargetRooms:         result.Criteria.TargetRooms,
			TargetArea:          result.Criteria.TargetArea,
			PreferredDistricts:  result.Criteria.PreferredDistricts,
		}
	}

	return resp, nil
}

