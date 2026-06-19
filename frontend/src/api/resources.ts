import type { Resource, ResourceAvailability, ResourceCategory, ResourceType } from "../types/resources";
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
