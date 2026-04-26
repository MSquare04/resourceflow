CREATE TABLE app.resource_categories (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.resource_types (
  id BIGSERIAL PRIMARY KEY,
  category_id BIGINT NOT NULL,
  code VARCHAR(50) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_resource_types_category
    FOREIGN KEY (category_id)
    REFERENCES app.resource_categories (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT
);

CREATE TABLE app.resources (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  category_id BIGINT NOT NULL,
  type_id BIGINT NOT NULL,
  department_id BIGINT NULL,
  location TEXT NULL,
  capacity BIGINT NULL,
  is_bookable BOOLEAN NOT NULL DEFAULT TRUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_resources_category
    FOREIGN KEY (category_id)
    REFERENCES app.resource_categories (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT fk_resources_type
    FOREIGN KEY (type_id)
    REFERENCES app.resource_types (id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,
  CONSTRAINT fk_resources_department
    FOREIGN KEY (department_id)
    REFERENCES app.departments (id)
    ON UPDATE CASCADE
    ON DELETE SET NULL,
  CONSTRAINT chk_resources_capacity_non_negative
    CHECK (capacity IS NULL OR capacity >= 0)
);

CREATE INDEX idx_resource_types_category_id ON app.resource_types (category_id);
CREATE INDEX idx_resources_category_id ON app.resources (category_id);
CREATE INDEX idx_resources_type_id ON app.resources (type_id);
CREATE INDEX idx_resources_department_id ON app.resources (department_id);
