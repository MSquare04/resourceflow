export interface Resource {
  id: number;
  name: string;
  description: string;
  category_id: number;
  type_id: number;
  department_id: number | null;
  location: string | null;
  capacity: number | null;
  is_bookable: boolean;
  is_active: boolean;
}

export interface ResourcePayload {
  name: string;
  description: string;
  category_id: number;
  type_id: number;
  department_id: number | null;
  location: string | null;
  capacity: number | null;
  is_bookable: boolean;
  is_active: boolean;
}

export interface ResourceCategory {
  id: number;
  code: string;
  name: string;
  description: string;
  is_active: boolean;
}

export interface ResourceType {
  id: number;
  category_id: number;
  code: string;
  name: string;
  description: string;
  is_active: boolean;
}

export interface ResourceAvailability {
  id: number;
  resource_id: number;
  start_at: string;
  end_at: string;
  created_at: string;
  updated_at: string;
}

export interface ResourceAvailabilityPayload {
  start_at: string;
  end_at: string;
}

export interface ResourceBusyInterval {
  start_at: string;
  end_at: string;
}
