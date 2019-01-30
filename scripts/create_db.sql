--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.14
-- Dumped by pg_dump version 9.5.14

-- Started on 2019-01-30 13:33:41 UTC

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 1 (class 3079 OID 12361)
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- TOC entry 2159 (class 0 OID 0)
-- Dependencies: 1
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET default_tablespace = '';

SET default_with_oids = false;

--
-- TOC entry 182 (class 1259 OID 16388)
-- Name: blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blocks (
    id integer NOT NULL,
    height integer NOT NULL,
    hash character(64) NOT NULL,
    header bytea NOT NULL,
    "timestamp" integer NOT NULL
);


--
-- TOC entry 181 (class 1259 OID 16386)
-- Name: blocks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.blocks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2160 (class 0 OID 0)
-- Dependencies: 181
-- Name: blocks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.blocks_id_seq OWNED BY public.blocks.id;


--
-- TOC entry 186 (class 1259 OID 16419)
-- Name: transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transactions (
    id bigint NOT NULL,
    hash character(64) NOT NULL,
    blob bytea NOT NULL,
    index_in_block integer NOT NULL,
    output_keys character(64)[] NOT NULL,
    output_indices bigint[],
    used_inputs bigint[] NOT NULL,
    "timestamp" integer NOT NULL,
    block_height integer NOT NULL
);


--
-- TOC entry 185 (class 1259 OID 16417)
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2161 (class 0 OID 0)
-- Dependencies: 185
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- TOC entry 184 (class 1259 OID 16399)
-- Name: wallets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wallets (
    id integer NOT NULL,
    secret_view_key character(64) NOT NULL,
    public_spend_key character(64) NOT NULL,
    last_checked_block_id integer,
    created_at integer NOT NULL
);


--
-- TOC entry 189 (class 1259 OID 16443)
-- Name: wallets_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wallets_blocks (
    id integer NOT NULL,
    wallet_id integer NOT NULL,
    block_id integer NOT NULL
);


--
-- TOC entry 188 (class 1259 OID 16441)
-- Name: wallets_blocks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.wallets_blocks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2162 (class 0 OID 0)
-- Dependencies: 188
-- Name: wallets_blocks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.wallets_blocks_id_seq OWNED BY public.wallets_blocks.id;


--
-- TOC entry 183 (class 1259 OID 16397)
-- Name: wallets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.wallets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2163 (class 0 OID 0)
-- Dependencies: 183
-- Name: wallets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.wallets_id_seq OWNED BY public.wallets.id;


--
-- TOC entry 187 (class 1259 OID 16431)
-- Name: wallets_outputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wallets_outputs (
    wallet_id integer NOT NULL,
    output integer NOT NULL,
    block_height integer NOT NULL
);


--
-- TOC entry 2009 (class 2604 OID 16391)
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blocks ALTER COLUMN id SET DEFAULT nextval('public.blocks_id_seq'::regclass);


--
-- TOC entry 2011 (class 2604 OID 2160451)
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- TOC entry 2010 (class 2604 OID 16402)
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets ALTER COLUMN id SET DEFAULT nextval('public.wallets_id_seq'::regclass);


--
-- TOC entry 2012 (class 2604 OID 16446)
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets_blocks ALTER COLUMN id SET DEFAULT nextval('public.wallets_blocks_id_seq'::regclass);


--
-- TOC entry 2017 (class 2606 OID 16396)
-- Name: blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (id);


--
-- TOC entry 2026 (class 2606 OID 2160453)
-- Name: transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- TOC entry 2031 (class 2606 OID 16448)
-- Name: wallets_blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets_blocks
    ADD CONSTRAINT wallets_blocks_pkey PRIMARY KEY (id);


--
-- TOC entry 2020 (class 2606 OID 16404)
-- Name: wallets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_pkey PRIMARY KEY (id);


--
-- TOC entry 2013 (class 1259 OID 16416)
-- Name: blocks_hash_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX blocks_hash_uindex ON public.blocks USING btree (hash);


--
-- TOC entry 2014 (class 1259 OID 16415)
-- Name: blocks_height_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX blocks_height_uindex ON public.blocks USING btree (height);


--
-- TOC entry 2015 (class 1259 OID 16414)
-- Name: blocks_id_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX blocks_id_uindex ON public.blocks USING btree (id);


--
-- TOC entry 2022 (class 1259 OID 16430)
-- Name: transactions_block_height_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX transactions_block_height_index ON public.transactions USING btree (block_height DESC);


--
-- TOC entry 2023 (class 1259 OID 16429)
-- Name: transactions_hash_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX transactions_hash_uindex ON public.transactions USING btree (hash);


--
-- TOC entry 2024 (class 1259 OID 2160454)
-- Name: transactions_id_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX transactions_id_uindex ON public.transactions USING btree (id);


--
-- TOC entry 2029 (class 1259 OID 16459)
-- Name: wallets_blocks_id_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX wallets_blocks_id_uindex ON public.wallets_blocks USING btree (id);


--
-- TOC entry 2032 (class 1259 OID 16460)
-- Name: wallets_blocks_wallet_id_block_id_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX wallets_blocks_wallet_id_block_id_uindex ON public.wallets_blocks USING btree (wallet_id, block_id);


--
-- TOC entry 2018 (class 1259 OID 16410)
-- Name: wallets_id_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX wallets_id_uindex ON public.wallets USING btree (id);


--
-- TOC entry 2027 (class 1259 OID 16439)
-- Name: wallets_outputs_block_height_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wallets_outputs_block_height_index ON public.wallets_outputs USING btree (block_height DESC);


--
-- TOC entry 2028 (class 1259 OID 16440)
-- Name: wallets_outputs_wallet_id_output_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX wallets_outputs_wallet_id_output_uindex ON public.wallets_outputs USING btree (wallet_id, output);


--
-- TOC entry 2021 (class 1259 OID 16411)
-- Name: wallets_secret_view_key_public_spend_key_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX wallets_secret_view_key_public_spend_key_uindex ON public.wallets USING btree (secret_view_key, public_spend_key);


--
-- TOC entry 2036 (class 2606 OID 16454)
-- Name: wallets_blocks_blocks_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets_blocks
    ADD CONSTRAINT wallets_blocks_blocks_id_fk FOREIGN KEY (block_id) REFERENCES public.blocks(id);


--
-- TOC entry 2033 (class 2606 OID 16405)
-- Name: wallets_blocks_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_blocks_id_fk FOREIGN KEY (last_checked_block_id) REFERENCES public.blocks(id);


--
-- TOC entry 2035 (class 2606 OID 16449)
-- Name: wallets_blocks_wallets_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets_blocks
    ADD CONSTRAINT wallets_blocks_wallets_id_fk FOREIGN KEY (wallet_id) REFERENCES public.wallets(id);


--
-- TOC entry 2034 (class 2606 OID 16434)
-- Name: wallets_outputs_wallets_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets_outputs
    ADD CONSTRAINT wallets_outputs_wallets_id_fk FOREIGN KEY (wallet_id) REFERENCES public.wallets(id);


--
-- TOC entry 2158 (class 0 OID 0)
-- Dependencies: 6
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: -
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM postgres;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2019-01-30 13:34:01 UTC

--
-- PostgreSQL database dump complete
--

