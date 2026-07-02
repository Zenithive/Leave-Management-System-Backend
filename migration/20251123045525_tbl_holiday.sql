-- +goose Up


-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS Tbl_Holiday (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,                    
    date DATE NOT NULL UNIQUE,              
    day TEXT NOT NULL,                     
    type TEXT NOT NULL,                     
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
