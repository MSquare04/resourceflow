import type { Resource, ResourceCategory, ResourceType } from "../types/resources";
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
