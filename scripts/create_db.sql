create database monero
	with owner postgres
;

create table blocks
(
	id serial not null
		constraint blocks_pkey
			primary key,
	height integer not null,
	hash char(64) not null,
	header bytea not null,
	timestamp integer not null
)
;

alter table blocks owner to postgres
;

create table wallets
(
	id serial not null
		constraint wallets_pkey
			primary key,
	secret_view_key char(64) not null,
	public_spend_key char(64) not null,
	last_checked_block_id integer
		constraint wallets_blocks_id_fk
			references blocks,
	created_at integer not null
)
;

alter table wallets owner to postgres
;

create unique index wallets_id_uindex
	on wallets (id)
;

create unique index wallets_secret_view_key_public_spend_key_uindex
	on wallets (secret_view_key, public_spend_key)
;

create unique index wallets_id_uindex_2
	on wallets (id)
;

create unique index wallets_secret_view_key_public_spend_key_uindex_2
	on wallets (secret_view_key, public_spend_key)
;

create unique index blocks_id_uindex
	on blocks (id)
;

create unique index blocks_height_uindex
	on blocks (height)
;

create unique index blocks_hash_uindex
	on blocks (hash)
;

create table transactions
(
	id serial not null
		constraint transactions_pkey
			primary key,
	hash char(64) not null,
	blob bytea not null,
	index_in_block integer not null,
	output_keys char(64) [] not null,
	output_indices integer[] not null,
	used_inputs integer[] not null,
	timestamp integer not null,
	block_height integer not null
)
;

alter table transactions owner to postgres
;

create unique index transactions_id_uindex
	on transactions (id)
;

create unique index transactions_hash_uindex
	on transactions (hash)
;

create index transactions_block_height_index
	on transactions (block_height desc)
;

create table wallets_blocks
(
	id serial not null
		constraint wallets_blocks_pkey
			primary key,
	wallet_id integer not null
		constraint wallets_blocks_wallets_id_fk
			references wallets,
	block_id integer not null
		constraint wallets_blocks_blocks_id_fk
			references blocks
)
;

alter table wallets_blocks owner to postgres
;

create unique index wallets_blocks_id_uindex
	on wallets_blocks (id)
;

create table wallets_outputs
(
	wallet_id integer not null
		constraint wallets_outputs_wallets_id_fk
			references wallets,
	output integer not null,
	block_height integer not null
)
;

alter table wallets_outputs owner to postgres
;

create index wallets_outputs_block_height_index
	on wallets_outputs (block_height desc)
;

