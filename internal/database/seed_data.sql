-- internal/database/seed_data.sql
-- Seed data for Rules Resolution Service
-- Run after schema.sql

-- Insert canonical workflow steps
INSERT INTO steps (key, name, description, position) VALUES
('title-search', 'Title Search', 'Research property ownership, liens, encumbrances, and tax status. Verify chain of title.', 1),
('file-complaint', 'File Complaint', 'Prepare and file the foreclosure complaint (judicial) or notice of default (non-judicial) with the court.', 2),
('serve-borrower', 'Serve Borrower', 'Serve the borrower and all named defendants with the complaint and summons via process server.', 3),
('obtain-judgment', 'Obtain Judgment', 'Obtain a judgment of foreclosure from the court authorizing the sale of the property.', 4),
('schedule-sale', 'Schedule Sale', 'Schedule the foreclosure sale date, coordinate publication requirements, and notify all parties.', 5),
('conduct-sale', 'Conduct Sale', 'Conduct the foreclosure auction, process bids, and file the certificate of sale.', 6)
ON CONFLICT (key) DO NOTHING;

-- Insert default values (abbreviated - include all from defaults.json)
INSERT INTO defaults (step_key, trait_key, value) VALUES
('title-search', 'slaHours', '720'),
('title-search', 'requiredDocuments', '["title_commitment","tax_certificate"]'),
('title-search', 'feeAmount', '35000'),
('title-search', 'feeAuthRequired', 'false'),
('title-search', 'assignedRole', '"processor"'),
('title-search', 'templateId', '"title-review-standard-v1"'),
('file-complaint', 'slaHours', '480'),
('file-complaint', 'requiredDocuments', '["complaint","summons","lis_pendens","cover_sheet"]'),
('file-complaint', 'feeAmount', '65000'),
('file-complaint', 'feeAuthRequired', 'false'),
('file-complaint', 'assignedRole', '"attorney"'),
('file-complaint', 'templateId', '"complaint-standard-v1"')
-- ... continue for all steps/traits
ON CONFLICT (step_key, trait_key) DO UPDATE SET value = EXCLUDED.value;

-- Insert override records (abbreviated - include all from overrides.json)
INSERT INTO overrides (id, step_key, trait_key, state, client, investor, case_type, value, effective_date, status, description, created_by) VALUES
('ovr-001', 'file-complaint', 'slaHours', 'FL', NULL, NULL, NULL, '360', '2025-01-01', 'active', 'Florida filing deadline — 15 days', 'admin@pearsonspecter.com'),
('ovr-002', 'file-complaint', 'requiredDocuments', 'FL', NULL, NULL, NULL, '["complaint","summons","lis_pendens","cover_sheet","verification_of_complaint"]', '2025-01-01', 'active', 'Florida requires verification of complaint', 'admin@pearsonspecter.com'),
('ovr-003', 'serve-borrower', 'slaHours', 'FL', NULL, NULL, NULL, '2160', '2025-01-01', 'active', 'Florida 90-day service window', 'admin@pearsonspecter.com'),
('ovr-020', 'file-complaint', 'slaHours', 'FL', 'Chase', NULL, NULL, '240', '2025-06-01', 'active', 'Chase in Florida — aggressive 10-day filing deadline', 'admin@pearsonspecter.com'),
('ovr-034', 'file-complaint', 'slaHours', 'FL', 'Chase', 'FHA', NULL, '168', '2025-09-01', 'active', 'FHA loans via Chase in Florida — 7-day filing deadline', 'admin@pearsonspecter.com'),
('ovr-047', 'file-complaint', 'templateId', 'FL', 'Chase', 'FannieMae', 'FC-Judicial', '"complaint-fl-chase-fnma-judicial-v3"', '2025-11-01', 'active', 'Fannie Mae judicial foreclosure via Chase in Florida', 'admin@pearsonspecter.com')
-- ... continue for all 49 overrides
ON CONFLICT (id) DO UPDATE SET
    value = EXCLUDED.value,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    description = EXCLUDED.description;