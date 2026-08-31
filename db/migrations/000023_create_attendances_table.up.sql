-- 1. TAMBAHKAN KOLOM USER_ID DAN DAILY_RATE PADA TABEL WORKERS
ALTER TABLE workers
ADD COLUMN user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
ADD COLUMN daily_rate NUMERIC(15, 2) NOT NULL DEFAULT 0;

CREATE INDEX idx_workers_user_id ON workers (user_id);

-- 2. TABEL ATTENDANCES (PENCATATAN PRESENSI/ABSENSI HARIAN)
CREATE TABLE IF NOT EXISTS attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    worker_id UUID NOT NULL REFERENCES workers (id) ON DELETE RESTRICT,
    payroll_id UUID NULL REFERENCES payrolls (id) ON DELETE SET NULL, -- Diisi saat dicairkan di rekap payroll
    attendance_date DATE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'present', -- 'present' (hadir), 'half_day' (setengah hari), 'permission' (izin), 'alpha' (tanpa keterangan)
    work_duration_index NUMERIC(3, 2) NOT NULL DEFAULT 1.00, -- 1.00 (full day), 0.50 (half day), 0.00 (permission/alpha)
    daily_rate NUMERIC(15, 2) NOT NULL DEFAULT 0, -- Tarif harian saat absensi dicatat
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0, -- work_duration_index * daily_rate
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users (id), -- Admin/Personalia yang mencatat absensi
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    CONSTRAINT unique_worker_attendance_date UNIQUE (worker_id, attendance_date) -- Mencegah absensi ganda pada tanggal yang sama
);

CREATE INDEX idx_attendances_worker_id ON attendances (worker_id);

CREATE INDEX idx_attendances_payroll_id ON attendances (payroll_id);

CREATE INDEX idx_attendances_date ON attendances (attendance_date);

CREATE INDEX idx_attendances_deleted_at ON attendances (deleted_at);