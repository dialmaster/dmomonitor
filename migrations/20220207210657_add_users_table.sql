-- +goose Up
-- +goose StatementBegin
create table users
 (
  id int not null auto_increment primary key,
  password_hash varchar(64),
  username varchar(64),
  created_at date,
  cloud_key varchar(64) 

 )engine=innodb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table users;
-- +goose StatementEnd
