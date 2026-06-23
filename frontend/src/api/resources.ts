import type {
  Resource,
  ResourceBusyInterval,
  ResourceCategory,
  ResourcePayload,
  ResourceUnavailability,
  ResourceUnavailabilityPayload,
  ResourceType,
} from "../types/resources";
import { apiRequest } from "./client";

export function listResources(): Promise<Resource[]> {
  return apiRequest<Resource[]>("/resources");
}

export function listResourceCategories(): Promise<ResourceCategory[]> {
  return apiRequest<ResourceCategory[]>("/resource-categories");
}

export function listResourceTypes(): Promise<ResourceType[]> {
  return apiRequest<ResourceType[]>("/resource-types");
}

export function getResource(id: number): Promise<Resource> {
  return apiRequest<Resource>(`/resources/${id}`);
}

export function listResourceBusyIntervals(resourceId: number): Promise<ResourceBusyInterval[]> {
  return apiRequest<ResourceBusyInterval[]>(`/resources/${resourceId}/busy-intervals`);
}

interface ResourceBusyIntervalsQuery {
  from?: string;
  to?: string;
}

export function listResourceBusyIntervalsInRange(
  resourceId: number,
  query: ResourceBusyIntervalsQuery,
): Promise<ResourceBusyInterval[]> {
  const searchParams = new URLSearchParams();

  if (query.from) {
    searchParams.set("from", query.from);
  }

  if (query.to) {
    searchParams.set("to", query.to);
  }

  const suffix = searchParams.toString();
  return apiRequest<ResourceBusyInterval[]>(`/resources/${resourceId}/busy-intervals${suffix ? `?${suffix}` : ""}`);
}

export function createResource(payload: ResourcePayload): Promise<Resource> {
  return apiRequest<Resource>("/resources", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateResource(id: number, payload: ResourcePayload): Promise<Resource> {
  return apiRequest<Resource>(`/resources/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function listResourceUnavailability(resourceId: number): Promise<ResourceUnavailability[]> {
  return apiRequest<ResourceUnavailability[]>(`/resources/${resourceId}/unavailability`);
}

export function createResourceUnavailability(
  resourceId: number,
  payload: ResourceUnavailabilityPayload,
): Promise<ResourceUnavailability> {
  return apiRequest<ResourceUnavailability>(`/resources/${resourceId}/unavailability`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateResourceUnavailability(
  resourceId: number,
  unavailabilityId: number,
  payload: ResourceUnavailabilityPayload,
): Promise<ResourceUnavailability> {
  return apiRequest<ResourceUnavailability>(`/resources/${resourceId}/unavailability/${unavailabilityId}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function deleteResourceUnavailability(
  resourceId: number,
  unavailabilityId: number,
): Promise<{ id: number; deleted: boolean }> {
  return apiRequest<{ id: number; deleted: boolean }>(`/resources/${resourceId}/unavailability/${unavailabilityId}`, {
    method: "DELETE",
  });
}
