package app

import "github.com/change-control-plane/change-control-plane/pkg/types"

func safeAuditEventsForResponse(items []types.AuditEvent) []types.AuditEvent {
	out := make([]types.AuditEvent, 0, len(items))
	for _, item := range items {
		item.Metadata = types.RedactMetadata(item.Metadata)
		out = append(out, item)
	}
	return out
}

func safeStatusEventsForResponse(items []types.StatusEvent) []types.StatusEvent {
	out := make([]types.StatusEvent, 0, len(items))
	for _, item := range items {
		item.Metadata = types.RedactMetadata(item.Metadata)
		out = append(out, item)
	}
	return out
}

func safeStatusEventQueryResultForResponse(result types.StatusEventQueryResult) types.StatusEventQueryResult {
	result.Events = safeStatusEventsForResponse(result.Events)
	result.Filters = types.RedactMetadata(result.Filters)
	return result
}

func safeRepositoryForResponse(item types.Repository) types.Repository {
	item.Metadata = types.RedactMetadata(item.Metadata)
	return item
}

func safeRepositoriesForResponse(items []types.Repository) []types.Repository {
	out := make([]types.Repository, 0, len(items))
	for _, item := range items {
		out = append(out, safeRepositoryForResponse(item))
	}
	return out
}

func safeDiscoveredResourceForResponse(item types.DiscoveredResource) types.DiscoveredResource {
	item.Metadata = types.RedactMetadata(item.Metadata)
	return item
}

func safeDiscoveredResourcesForResponse(items []types.DiscoveredResource) []types.DiscoveredResource {
	out := make([]types.DiscoveredResource, 0, len(items))
	for _, item := range items {
		out = append(out, safeDiscoveredResourceForResponse(item))
	}
	return out
}

func safeGraphRelationshipForResponse(item types.GraphRelationship) types.GraphRelationship {
	item.Metadata = types.RedactMetadata(item.Metadata)
	return item
}

func safeGraphRelationshipsForResponse(items []types.GraphRelationship) []types.GraphRelationship {
	out := make([]types.GraphRelationship, 0, len(items))
	for _, item := range items {
		out = append(out, safeGraphRelationshipForResponse(item))
	}
	return out
}

func safeRolloutExecutionForResponse(item types.RolloutExecution) types.RolloutExecution {
	item.Metadata = types.RedactMetadata(item.Metadata)
	return item
}

func safeVerificationResultForResponse(item types.VerificationResult) types.VerificationResult {
	item.Metadata = types.RedactMetadata(item.Metadata)
	item.TechnicalSignalSummary = types.RedactMetadata(item.TechnicalSignalSummary)
	item.BusinessSignalSummary = types.RedactMetadata(item.BusinessSignalSummary)
	return item
}

func safeSignalSnapshotForResponse(item types.SignalSnapshot) types.SignalSnapshot {
	item.Metadata = types.RedactMetadata(item.Metadata)
	return item
}

func safeRolloutExecutionDetailForResponse(detail types.RolloutExecutionDetail) types.RolloutExecutionDetail {
	detail.Execution = safeRolloutExecutionForResponse(detail.Execution)
	for idx := range detail.VerificationResults {
		detail.VerificationResults[idx] = safeVerificationResultForResponse(detail.VerificationResults[idx])
	}
	for idx := range detail.SignalSnapshots {
		detail.SignalSnapshots[idx] = safeSignalSnapshotForResponse(detail.SignalSnapshots[idx])
	}
	detail.Timeline = safeAuditEventsForResponse(detail.Timeline)
	detail.StatusTimeline = safeStatusEventsForResponse(detail.StatusTimeline)
	return detail
}
