CREATE TABLE expense_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'HPP' atau 'OPEX'
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_expense_categories_deleted_at ON expense_categories(deleted_at);

CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    expense_category_id UUID NOT NULL REFERENCES expense_categories(id),
    batch_po_id UUID REFERENCES batch_pos(id), -- Bisa NULL untuk OPEX
    title VARCHAR(255) NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    expense_date DATE NOT NULL,
    notes TEXT,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_expenses_category_id ON expenses(expense_category_id);
CREATE INDEX idx_expenses_batch_po_id ON expenses(batch_po_id);
CREATE INDEX idx_expenses_expense_date ON expenses(expense_date);
CREATE INDEX idx_expenses_deleted_at ON expenses(deleted_at);