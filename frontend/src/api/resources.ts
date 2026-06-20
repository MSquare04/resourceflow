import type {
  Resource,
  ResourceAvailability,
  ResourceAvailabilityPayload,
  ResourceBusyInterval,
  ResourceCategory,
  ResourcePayload,
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

export function listResourceAvailability(resourceId: number): Promise<ResourceAvailability[]> {
  return apiRequest<ResourceAvailability[]>(`/resources/${resourceId}/availability`);
}

export function listResourceBusyIntervals(resourceId: number): Promise<ResourceBusyInterval[]> {
  return apiRequest<ResourceBusyInterval[]>(`/resources/${resourceId}/busy-intervals`);
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

export function createResourceAvailability(
  resourceId: number,
  payload: ResourceAvailabilityPayload,
): Promise<ResourceAvailability> {
  return apiRequest<ResourceAvailability>(`/resources/${resourceId}/availability`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateResourceAvailability(
  resourceId: number,
  availabilityId: number,
  payload: ResourceAvailabilityPayload,
): Promise<ResourceAvailability> {
  return apiRequest<ResourceAvailability>(`/resources/${resourceId}/availability/${availabilityId}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function deleteResourceAvailability(resourceId: number, availabilityId: number): Promise<{ id: number; deleted: boolean }> {
  return apiRequest<{ id: number; deleted: boolean }>(`/resources/${resourceId}/availability/${availabilityId}`, {
    method: "DELETE",
  });
}
