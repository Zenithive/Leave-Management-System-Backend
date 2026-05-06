-- +goose Up
ALTER TABLE Tbl_Company_Settings
    ADD COLUMN birthday_message_template TEXT NOT NULL DEFAULT 'Happy Birthday {name}! 🎉 Wishing you a wonderful day and a fantastic year ahead!';

-- +goose Down
ALTER TABLE Tbl_Company_Settings DROP COLUMN IF EXISTS birthday_message_template;
