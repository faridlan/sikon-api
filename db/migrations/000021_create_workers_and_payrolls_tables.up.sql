-- 1. TABEL WORKERS (MASTER PEKERJA LAPANGAN)
CREATE TABLE IF NOT EXISTS workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    role VARCHAR(50) NOT NULL, -- 'tailor' (penjahit), 'cutter' (pemotong), 'finishing', 'helper'
    salary_type VARCHAR(50) NOT NULL, -- 'piece_rate' (borongan per pcs), 'daily' (harian), 'monthly' (bulanan)
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'inactive'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX idx_workers_deleted_at ON workers(deleted_at);
CREATE INDEX idx_workers_role ON workers(role);

-- 2. TABEL PAYROLLS (REKAP PENGGAJIAN)
-- Catatan: Dibuat sebelum work_logs agar constraint FK payroll_id di work_logs bisa mereferensikannya
CREATE TABLE IF NOT EXISTS payrolls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_number VARCHAR(100) NOT NULL UNIQUE, -- e.g., 'PAY-202608-001'
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- 'draft', 'approved', 'paid'
    expense_id UUID NULL REFERENCES expenses(id) ON DELETE SET NULL, -- Ref ke auto-generated expense
    created_by UUID NOT NULL REFERENCES users(id), -- Accounting / HR
    paid_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX idx_payrolls_status ON payrolls(status);
CREATE INDEX idx_payrolls_deleted_at ON payrolls(deleted_at);

-- 3. TABEL WORK_LOGS (PENCATATAN HASIL BORONGAN OLEH PERSONALIA)
CREATE TABLE IF NOT EXISTS work_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id UUID NOT NULL REFERENCES workers(id) ON DELETE RESTRICT,
    batch_po_id UUID NULL REFERENCES batch_pos(id) ON DELETE SET NULL, -- Linked ke Batch PO untuk hitung HPP
    payroll_id UUID NULL REFERENCES payrolls(id) ON DELETE SET NULL, -- Will be filled when grouped into a payroll period
    job_type VARCHAR(50) NOT NULL, -- 'jahit', 'potong', 'bordir', 'finishing'
    qty INT NOT NULL DEFAULT 1,
    rate_per_qty NUMERIC(15, 2) NOT NULL DEFAULT 0, -- Tarif per pcs (bisa override jika ada kustomisasi)
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0, -- qty * rate_per_qty
    work_date DATE NOT NULL,
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id), -- ID Personalia / Admin
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX idx_work_logs_worker_id ON work_logs(worker_id);
CREATE INDEX idx_work_logs_batch_po_id ON work_logs(batch_po_id);
CREATE INDEX idx_work_logs_payroll_id ON work_logs(payroll_id);
CREATE INDEX idx_work_logs_work_date ON work_logs(work_date);