-- +goose Up
-- +goose StatementBegin
create table receiving_addresses
 (
  id int not null auto_increment primary key,
  user_id int,
  display_name varchar(64),
  receiving_address varchar(64)
 )engine=innodb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE receiving_addresses;
-- +goose StatementEnd
