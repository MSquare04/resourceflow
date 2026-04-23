DELETE FROM app.roles
WHERE code IN ('admin', 'manager', 'employee', 'hr', 'interviewer');
