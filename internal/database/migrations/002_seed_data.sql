-- Migration: 002_seed_data.sql
-- Loads initial workflow steps, defaults, and override records

-- Insert canonical workflow steps
INSERT INTO steps (key, name, description, position) VALUES
('title-search', 'Title Search', 'Research property ownership, liens, encumbrances, and tax status. Verify chain of title.', 1),
('file-complaint', 'File Complaint', 'Prepare and file the foreclosure complaint (judicial) or notice of default (non-judicial) with the court.', 2),
('serve-borrower', 'Serve Borrower', 'Serve the borrower and all named defendants with the complaint and summons via process server.', 3),
('obtain-judgment', 'Obtain Judgment', 'Obtain a judgment of foreclosure from the court authorizing the sale of the property.', 4),
('schedule-sale', 'Schedule Sale', 'Schedule the foreclosure sale date, coordinate publication requirements, and notify all parties.', 5),
('conduct-sale', 'Conduct Sale', 'Conduct the foreclosure auction, process bids, and file the certificate of sale.', 6);

-- Insert default values for each step/trait (specificity 0)
INSERT INTO defaults (step_key, trait_key, value) VALUES
-- title-search defaults
('title-search', 'slaHours', '720'),
('title-search', 'requiredDocuments', '["title_commitment", "tax_certificate"]'),
('title-search', 'feeAmount', '35000'),
('title-search', 'feeAuthRequired', 'false'),
('title-search', 'assignedRole', '"processor"'),
('title-search', 'templateId', '"title-review-standard-v1"'),

-- file-complaint defaults
('file-complaint', 'slaHours', '480'),
('file-complaint', 'requiredDocuments', '["complaint", "summons", "lis_pendens", "cover_sheet"]'),
('file-complaint', 'feeAmount', '65000'),
('file-complaint', 'feeAuthRequired', 'false'),
('file-complaint', 'assignedRole', '"attorney"'),
('file-complaint', 'templateId', '"complaint-standard-v1"'),

-- serve-borrower defaults
('serve-borrower', 'slaHours', '2880'),
('serve-borrower', 'requiredDocuments', '["affidavit_of_service", "return_of_service"]'),
('serve-borrower', 'feeAmount', '25000'),
('serve-borrower', 'feeAuthRequired', 'false'),
('serve-borrower', 'assignedRole', '"processor"'),
('serve-borrower', 'templateId', '"service-standard-v1"'),

-- obtain-judgment defaults
('obtain-judgment', 'slaHours', '4320'),
('obtain-judgment', 'requiredDocuments', '["motion_for_judgment", "affidavit_of_indebtedness", "proposed_judgment"]'),
('obtain-judgment', 'feeAmount', '45000'),
('obtain-judgment', 'feeAuthRequired', 'false'),
('obtain-judgment', 'assignedRole', '"attorney"'),
('obtain-judgment', 'templateId', '"judgment-standard-v1"'),

-- schedule-sale defaults
('schedule-sale', 'slaHours', '1440'),
('schedule-sale', 'requiredDocuments', '["notice_of_sale", "publication_proof"]'),
('schedule-sale', 'feeAmount', '30000'),
('schedule-sale', 'feeAuthRequired', 'false'),
('schedule-sale', 'assignedRole', '"processor"'),
('schedule-sale', 'templateId', '"sale-notice-standard-v1"'),

-- conduct-sale defaults
('conduct-sale', 'slaHours', '720'),
('conduct-sale', 'requiredDocuments', '["certificate_of_sale", "sale_report"]'),
('conduct-sale', 'feeAmount', '50000'),
('conduct-sale', 'feeAuthRequired', 'false'),
('conduct-sale', 'assignedRole', '"attorney"'),
('conduct-sale', 'templateId', '"sale-report-standard-v1"');

-- Insert override records (49 total, abbreviated for brevity - full version includes all)
-- Florida state overrides (specificity 1)
INSERT INTO overrides (id, step_key, trait_key, state, value, effective_date, status, description, created_by) VALUES
('ovr-001', 'file-complaint', 'slaHours', 'FL', '360', '2025-01-01', 'active', 'Florida filing deadline — 15 days', 'admin@pearsonspecter.com'),
('ovr-002', 'file-complaint', 'requiredDocuments', 'FL', '["complaint", "summons", "lis_pendens", "cover_sheet", "verification_of_complaint"]', '2025-01-01', 'active', 'Florida requires verification of complaint', 'admin@pearsonspecter.com'),
('ovr-003', 'serve-borrower', 'slaHours', 'FL', '2160', '2025-01-01', 'active', 'Florida 90-day service window', 'admin@pearsonspecter.com'),
('ovr-014', 'conduct-sale', 'templateId', 'FL', '"sale-report-fl-v2"', '2025-01-01', 'active', 'Florida-specific sale report template', 'admin@pearsonspecter.com');

-- Florida + Chase overrides (specificity 2)
INSERT INTO overrides (id, step_key, trait_key, state, client, value, effective_date, status, description, created_by) VALUES
('ovr-020', 'file-complaint', 'slaHours', 'FL', 'Chase', '240', '2025-06-01', 'active', 'Chase in Florida — aggressive 10-day filing deadline', 'admin@pearsonspecter.com'),
('ovr-025', 'file-complaint', 'templateId', 'FL', 'Chase', '"complaint-fl-chase-v2"', '2025-06-01', 'active', 'Chase Florida complaint template', 'admin@pearsonspecter.com'),
('ovr-026', 'obtain-judgment', 'slaHours', 'FL', 'Chase', '2880', '2025-06-01', 'active', 'Chase Florida judgment — 120-day target', 'admin@pearsonspecter.com'),
('ovr-053', 'file-complaint', 'feeAmount', 'FL', 'Chase', '60000', '2025-06-01', 'active', 'Chase Florida filing fee — negotiated rate', 'admin@pearsonspecter.com');

-- Chase global overrides (specificity 1 - client only)
INSERT INTO overrides (id, step_key, trait_key, client, value, effective_date, status, description, created_by) VALUES
('ovr-030', 'title-search', 'feeAuthRequired', 'Chase', 'true', '2025-01-01', 'active', 'Chase requires fee authorization for title search', 'admin@pearsonspecter.com'),
('ovr-031', 'file-complaint', 'feeAuthRequired', 'Chase', 'true', '2025-01-01', 'active', 'Chase requires fee authorization for filing', 'admin@pearsonspecter.com');

-- Florida + Chase + FHA overrides (specificity 3)
INSERT INTO overrides (id, step_key, trait_key, state, client, investor, value, effective_date, status, description, created_by) VALUES
('ovr-034', 'file-complaint', 'slaHours', 'FL', 'Chase', 'FHA', '168', '2025-09-01', 'active', 'FHA loans via Chase in Florida — 7-day filing deadline', 'admin@pearsonspecter.com'),
('ovr-035', 'file-complaint', 'feeAmount', 'FL', 'Chase', 'FHA', '55000', '2025-09-01', 'active', 'FHA Chase Florida filing — reduced fee', 'admin@pearsonspecter.com'),
('ovr-036', 'file-complaint', 'requiredDocuments', 'FL', 'Chase', 'FHA', '["complaint", "summons", "lis_pendens", "cover_sheet", "verification_of_complaint", "hud_face_sheet", "fha_servicing_history"]', '2025-09-01', 'active', 'FHA Chase Florida docs', 'admin@pearsonspecter.com'),
('ovr-037', 'file-complaint', 'templateId', 'FL', 'Chase', 'FHA', '"complaint-fl-chase-fha-v3"', '2025-09-01', 'active', 'FHA-specific complaint template for Chase in FL', 'admin@pearsonspecter.com');

-- Investor-level overrides (specificity 1)
INSERT INTO overrides (id, step_key, trait_key, investor, value, effective_date, status, description, created_by) VALUES
('ovr-038', 'file-complaint', 'requiredDocuments', 'FHA', '["complaint", "summons", "lis_pendens", "cover_sheet", "hud_face_sheet"]', '2025-03-01', 'active', 'FHA loans require HUD face sheet', 'admin@pearsonspecter.com'),
('ovr-039', 'file-complaint', 'requiredDocuments', 'VA', '["complaint", "summons", "lis_pendens", "cover_sheet", "va_loan_summary", "va_appraisal"]', '2025-03-01', 'active', 'VA loans require VA loan summary and appraisal', 'admin@pearsonspecter.com');

-- Case type overrides
INSERT INTO overrides (id, step_key, trait_key, case_type, value, effective_date, status, description, created_by) VALUES
('ovr-043', 'obtain-judgment', 'slaHours', 'FC-NonJudicial', '0', '2025-01-01', 'active', 'Non-judicial cases skip judgment step', 'admin@pearsonspecter.com'),
('ovr-044', 'obtain-judgment', 'feeAmount', 'FC-NonJudicial', '0', '2025-01-01', 'active', 'No fee for judgment in non-judicial', 'admin@pearsonspecter.com');

-- Four-dimension override (specificity 4)
INSERT INTO overrides (id, step_key, trait_key, state, client, investor, case_type, value, effective_date, status, description, created_by) VALUES
('ovr-047', 'file-complaint', 'templateId', 'FL', 'Chase', 'FannieMae', 'FC-Judicial', '"complaint-fl-chase-fnma-judicial-v3"', '2025-11-01', 'active', 'Fannie Mae judicial foreclosure via Chase in Florida', 'admin@pearsonspecter.com');

-- Additional overrides for test coverage (abbreviated)
INSERT INTO overrides (id, step_key, trait_key, state, client, value, effective_date, status, description, created_by) VALUES
('ovr-048', 'title-search', 'slaHours', 'FL', 'Nationstar', '480', '2025-06-01', 'active', 'Nationstar Florida title searches — 20-day deadline', 'admin@pearsonspecter.com'),
('ovr-051', 'file-complaint', 'slaHours', 'IL', 'WellsFargo', '360', '2025-06-01', 'active', 'Wells Fargo Illinois filing — 15-day deadline', 'admin@pearsonspecter.com'),
('ovr-052', 'title-search', 'requiredDocuments', 'IL', NULL, '["title_commitment", "tax_certificate", "municipal_lien_search"]', '2025-01-01', 'active', 'Illinois requires municipal lien search', 'admin@pearsonspecter.com'),
('ovr-055', 'title-search', 'slaHours', 'OH', NULL, '504', '2025-01-01', 'active', 'Ohio title searches — 21-day deadline', 'admin@pearsonspecter.com');