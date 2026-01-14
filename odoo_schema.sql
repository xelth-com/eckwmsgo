--
-- PostgreSQL database dump
--

\restrict bCjjil46YuV2CH1pXFVzyU8a5hjJit6d4xaxHPzLw45Acug6Uw1ficJDKghSA8a

-- Dumped from database version 14.19 (Ubuntu 14.19-1.pgdg24.04+1)
-- Dumped by pg_dump version 14.19 (Ubuntu 14.19-1.pgdg24.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: product_product; Type: TABLE; Schema: public; Owner: dimi
--

CREATE TABLE public.product_product (
    id integer NOT NULL,
    product_tmpl_id integer NOT NULL,
    create_uid integer,
    write_uid integer,
    default_code character varying,
    barcode character varying,
    combination_indices character varying,
    volume numeric,
    weight numeric,
    active boolean,
    can_image_variant_1024_be_zoomed boolean,
    write_date timestamp without time zone,
    create_date timestamp without time zone,
    lot_properties_definition jsonb
);


ALTER TABLE public.product_product OWNER TO dimi;

--
-- Name: TABLE product_product; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON TABLE public.product_product IS 'Product Variant';


--
-- Name: COLUMN product_product.product_tmpl_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.product_tmpl_id IS 'Product Template';


--
-- Name: COLUMN product_product.create_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.create_uid IS 'Created by';


--
-- Name: COLUMN product_product.write_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.write_uid IS 'Last Updated by';


--
-- Name: COLUMN product_product.default_code; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.default_code IS 'Internal Reference';


--
-- Name: COLUMN product_product.barcode; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.barcode IS 'Barcode';


--
-- Name: COLUMN product_product.combination_indices; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.combination_indices IS 'Combination Indices';


--
-- Name: COLUMN product_product.volume; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.volume IS 'Volume';


--
-- Name: COLUMN product_product.weight; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.weight IS 'Weight';


--
-- Name: COLUMN product_product.active; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.active IS 'Active';


--
-- Name: COLUMN product_product.can_image_variant_1024_be_zoomed; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.can_image_variant_1024_be_zoomed IS 'Can Variant Image 1024 be zoomed';


--
-- Name: COLUMN product_product.write_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.write_date IS 'Write Date';


--
-- Name: COLUMN product_product.create_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.create_date IS 'Created on';


--
-- Name: COLUMN product_product.lot_properties_definition; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.product_product.lot_properties_definition IS 'Lot Properties';


--
-- Name: product_product_id_seq; Type: SEQUENCE; Schema: public; Owner: dimi
--

CREATE SEQUENCE public.product_product_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.product_product_id_seq OWNER TO dimi;

--
-- Name: product_product_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: dimi
--

ALTER SEQUENCE public.product_product_id_seq OWNED BY public.product_product.id;


--
-- Name: stock_location; Type: TABLE; Schema: public; Owner: dimi
--

CREATE TABLE public.stock_location (
    id integer NOT NULL,
    location_id integer,
    posx integer,
    posy integer,
    posz integer,
    company_id integer,
    removal_strategy_id integer,
    cyclic_inventory_frequency integer,
    warehouse_id integer,
    storage_category_id integer,
    create_uid integer,
    write_uid integer,
    name character varying NOT NULL,
    complete_name character varying,
    usage character varying NOT NULL,
    parent_path character varying,
    barcode character varying,
    last_inventory_date date,
    next_inventory_date date,
    comment text,
    active boolean,
    scrap_location boolean,
    return_location boolean,
    replenish_location boolean,
    create_date timestamp without time zone,
    write_date timestamp without time zone,
    CONSTRAINT stock_location_inventory_freq_nonneg CHECK ((cyclic_inventory_frequency >= 0))
);


ALTER TABLE public.stock_location OWNER TO dimi;

--
-- Name: TABLE stock_location; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON TABLE public.stock_location IS 'Inventory Locations';


--
-- Name: COLUMN stock_location.location_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.location_id IS 'Parent Location';


--
-- Name: COLUMN stock_location.posx; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.posx IS 'Corridor (X)';


--
-- Name: COLUMN stock_location.posy; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.posy IS 'Shelves (Y)';


--
-- Name: COLUMN stock_location.posz; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.posz IS 'Height (Z)';


--
-- Name: COLUMN stock_location.company_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.company_id IS 'Company';


--
-- Name: COLUMN stock_location.removal_strategy_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.removal_strategy_id IS 'Removal Strategy';


--
-- Name: COLUMN stock_location.cyclic_inventory_frequency; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.cyclic_inventory_frequency IS 'Inventory Frequency (Days)';


--
-- Name: COLUMN stock_location.warehouse_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.warehouse_id IS 'Warehouse';


--
-- Name: COLUMN stock_location.storage_category_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.storage_category_id IS 'Storage Category';


--
-- Name: COLUMN stock_location.create_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.create_uid IS 'Created by';


--
-- Name: COLUMN stock_location.write_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.write_uid IS 'Last Updated by';


--
-- Name: COLUMN stock_location.name; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.name IS 'Location Name';


--
-- Name: COLUMN stock_location.complete_name; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.complete_name IS 'Full Location Name';


--
-- Name: COLUMN stock_location.usage; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.usage IS 'Location Type';


--
-- Name: COLUMN stock_location.parent_path; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.parent_path IS 'Parent Path';


--
-- Name: COLUMN stock_location.barcode; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.barcode IS 'Barcode';


--
-- Name: COLUMN stock_location.last_inventory_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.last_inventory_date IS 'Last Effective Inventory';


--
-- Name: COLUMN stock_location.next_inventory_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.next_inventory_date IS 'Next Expected Inventory';


--
-- Name: COLUMN stock_location.comment; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.comment IS 'Additional Information';


--
-- Name: COLUMN stock_location.active; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.active IS 'Active';


--
-- Name: COLUMN stock_location.scrap_location; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.scrap_location IS 'Is a Scrap Location?';


--
-- Name: COLUMN stock_location.return_location; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.return_location IS 'Is a Return Location?';


--
-- Name: COLUMN stock_location.replenish_location; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.replenish_location IS 'Replenish Location';


--
-- Name: COLUMN stock_location.create_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.create_date IS 'Created on';


--
-- Name: COLUMN stock_location.write_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_location.write_date IS 'Last Updated on';


--
-- Name: CONSTRAINT stock_location_inventory_freq_nonneg ON stock_location; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON CONSTRAINT stock_location_inventory_freq_nonneg ON public.stock_location IS 'check(cyclic_inventory_frequency >= 0)';


--
-- Name: stock_quant; Type: TABLE; Schema: public; Owner: dimi
--

CREATE TABLE public.stock_quant (
    id integer NOT NULL,
    product_id integer NOT NULL,
    company_id integer,
    location_id integer NOT NULL,
    storage_category_id integer,
    lot_id integer,
    package_id integer,
    owner_id integer,
    user_id integer,
    create_uid integer,
    write_uid integer,
    inventory_date date,
    quantity numeric,
    reserved_quantity numeric NOT NULL,
    inventory_quantity numeric,
    inventory_diff_quantity numeric,
    inventory_quantity_set boolean,
    in_date timestamp without time zone NOT NULL,
    create_date timestamp without time zone,
    write_date timestamp without time zone
);


ALTER TABLE public.stock_quant OWNER TO dimi;

--
-- Name: TABLE stock_quant; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON TABLE public.stock_quant IS 'Quants';


--
-- Name: COLUMN stock_quant.product_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.product_id IS 'Product';


--
-- Name: COLUMN stock_quant.company_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.company_id IS 'Company';


--
-- Name: COLUMN stock_quant.location_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.location_id IS 'Location';


--
-- Name: COLUMN stock_quant.storage_category_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.storage_category_id IS 'Storage Category';


--
-- Name: COLUMN stock_quant.lot_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.lot_id IS 'Lot/Serial Number';


--
-- Name: COLUMN stock_quant.package_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.package_id IS 'Package';


--
-- Name: COLUMN stock_quant.owner_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.owner_id IS 'Owner';


--
-- Name: COLUMN stock_quant.user_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.user_id IS 'Assigned To';


--
-- Name: COLUMN stock_quant.create_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.create_uid IS 'Created by';


--
-- Name: COLUMN stock_quant.write_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.write_uid IS 'Last Updated by';


--
-- Name: COLUMN stock_quant.inventory_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.inventory_date IS 'Scheduled Date';


--
-- Name: COLUMN stock_quant.quantity; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.quantity IS 'Quantity';


--
-- Name: COLUMN stock_quant.reserved_quantity; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.reserved_quantity IS 'Reserved Quantity';


--
-- Name: COLUMN stock_quant.inventory_quantity; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.inventory_quantity IS 'Counted Quantity';


--
-- Name: COLUMN stock_quant.inventory_diff_quantity; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.inventory_diff_quantity IS 'Difference';


--
-- Name: COLUMN stock_quant.inventory_quantity_set; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.inventory_quantity_set IS 'Inventory Quantity Set';


--
-- Name: COLUMN stock_quant.in_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.in_date IS 'Incoming Date';


--
-- Name: COLUMN stock_quant.create_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.create_date IS 'Created on';


--
-- Name: COLUMN stock_quant.write_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_quant.write_date IS 'Last Updated on';


--
-- Name: stock_location_id_seq; Type: SEQUENCE; Schema: public; Owner: dimi
--

CREATE SEQUENCE public.stock_location_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.stock_location_id_seq OWNER TO dimi;

--
-- Name: stock_location_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: dimi
--

ALTER SEQUENCE public.stock_location_id_seq OWNED BY public.stock_location.id;


--
-- Name: stock_move_line; Type: TABLE; Schema: public; Owner: dimi
--

CREATE TABLE public.stock_move_line (
    id integer NOT NULL,
    picking_id integer,
    move_id integer,
    company_id integer NOT NULL,
    product_id integer,
    product_uom_id integer NOT NULL,
    package_id integer,
    package_level_id integer,
    lot_id integer,
    result_package_id integer,
    owner_id integer,
    location_id integer NOT NULL,
    location_dest_id integer NOT NULL,
    create_uid integer,
    write_uid integer,
    product_category_name character varying,
    lot_name character varying,
    state character varying,
    reference character varying,
    description_picking text,
    quantity numeric,
    quantity_product_uom numeric,
    picked boolean,
    date timestamp without time zone NOT NULL,
    create_date timestamp without time zone,
    write_date timestamp without time zone
);


ALTER TABLE public.stock_move_line OWNER TO dimi;

--
-- Name: TABLE stock_move_line; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON TABLE public.stock_move_line IS 'Product Moves (Stock Move Line)';


--
-- Name: COLUMN stock_move_line.picking_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.picking_id IS 'Transfer';


--
-- Name: COLUMN stock_move_line.move_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.move_id IS 'Stock Operation';


--
-- Name: COLUMN stock_move_line.company_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.company_id IS 'Company';


--
-- Name: COLUMN stock_move_line.product_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.product_id IS 'Product';


--
-- Name: COLUMN stock_move_line.product_uom_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.product_uom_id IS 'Unit of Measure';


--
-- Name: COLUMN stock_move_line.package_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.package_id IS 'Source Package';


--
-- Name: COLUMN stock_move_line.package_level_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.package_level_id IS 'Package Level';


--
-- Name: COLUMN stock_move_line.lot_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.lot_id IS 'Lot/Serial Number';


--
-- Name: COLUMN stock_move_line.result_package_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.result_package_id IS 'Destination Package';


--
-- Name: COLUMN stock_move_line.owner_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.owner_id IS 'From Owner';


--
-- Name: COLUMN stock_move_line.location_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.location_id IS 'From';


--
-- Name: COLUMN stock_move_line.location_dest_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.location_dest_id IS 'To';


--
-- Name: COLUMN stock_move_line.create_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.create_uid IS 'Created by';


--
-- Name: COLUMN stock_move_line.write_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.write_uid IS 'Last Updated by';


--
-- Name: COLUMN stock_move_line.product_category_name; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.product_category_name IS 'Product Category';


--
-- Name: COLUMN stock_move_line.lot_name; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.lot_name IS 'Lot/Serial Number Name';


--
-- Name: COLUMN stock_move_line.state; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.state IS 'Status';


--
-- Name: COLUMN stock_move_line.reference; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.reference IS 'Reference';


--
-- Name: COLUMN stock_move_line.description_picking; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.description_picking IS 'Description picking';


--
-- Name: COLUMN stock_move_line.quantity; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.quantity IS 'Quantity';


--
-- Name: COLUMN stock_move_line.quantity_product_uom; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.quantity_product_uom IS 'Quantity in Product UoM';


--
-- Name: COLUMN stock_move_line.picked; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.picked IS 'Picked';


--
-- Name: COLUMN stock_move_line.date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.date IS 'Date';


--
-- Name: COLUMN stock_move_line.create_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.create_date IS 'Created on';


--
-- Name: COLUMN stock_move_line.write_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_move_line.write_date IS 'Last Updated on';


--
-- Name: stock_move_line_id_seq; Type: SEQUENCE; Schema: public; Owner: dimi
--

CREATE SEQUENCE public.stock_move_line_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.stock_move_line_id_seq OWNER TO dimi;

--
-- Name: stock_move_line_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: dimi
--

ALTER SEQUENCE public.stock_move_line_id_seq OWNED BY public.stock_move_line.id;


--
-- Name: stock_picking; Type: TABLE; Schema: public; Owner: dimi
--

CREATE TABLE public.stock_picking (
    id integer NOT NULL,
    backorder_id integer,
    return_id integer,
    group_id integer,
    location_id integer NOT NULL,
    location_dest_id integer NOT NULL,
    picking_type_id integer NOT NULL,
    partner_id integer,
    company_id integer,
    user_id integer,
    owner_id integer,
    create_uid integer,
    write_uid integer,
    name character varying,
    origin character varying,
    move_type character varying NOT NULL,
    state character varying,
    priority character varying,
    picking_properties jsonb,
    note text,
    has_deadline_issue boolean,
    printed boolean,
    is_locked boolean,
    scheduled_date timestamp without time zone,
    date_deadline timestamp without time zone,
    date timestamp without time zone,
    date_done timestamp without time zone,
    create_date timestamp without time zone,
    write_date timestamp without time zone
);


ALTER TABLE public.stock_picking OWNER TO dimi;

--
-- Name: TABLE stock_picking; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON TABLE public.stock_picking IS 'Transfer';


--
-- Name: COLUMN stock_picking.backorder_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.backorder_id IS 'Back Order of';


--
-- Name: COLUMN stock_picking.return_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.return_id IS 'Return of';


--
-- Name: COLUMN stock_picking.group_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.group_id IS 'Procurement Group';


--
-- Name: COLUMN stock_picking.location_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.location_id IS 'Source Location';


--
-- Name: COLUMN stock_picking.location_dest_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.location_dest_id IS 'Destination Location';


--
-- Name: COLUMN stock_picking.picking_type_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.picking_type_id IS 'Operation Type';


--
-- Name: COLUMN stock_picking.partner_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.partner_id IS 'Contact';


--
-- Name: COLUMN stock_picking.company_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.company_id IS 'Company';


--
-- Name: COLUMN stock_picking.user_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.user_id IS 'Responsible';


--
-- Name: COLUMN stock_picking.owner_id; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.owner_id IS 'Assign Owner';


--
-- Name: COLUMN stock_picking.create_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.create_uid IS 'Created by';


--
-- Name: COLUMN stock_picking.write_uid; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.write_uid IS 'Last Updated by';


--
-- Name: COLUMN stock_picking.name; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.name IS 'Reference';


--
-- Name: COLUMN stock_picking.origin; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.origin IS 'Source Document';


--
-- Name: COLUMN stock_picking.move_type; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.move_type IS 'Shipping Policy';


--
-- Name: COLUMN stock_picking.state; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.state IS 'Status';


--
-- Name: COLUMN stock_picking.priority; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.priority IS 'Priority';


--
-- Name: COLUMN stock_picking.picking_properties; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.picking_properties IS 'Properties';


--
-- Name: COLUMN stock_picking.note; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.note IS 'Notes';


--
-- Name: COLUMN stock_picking.has_deadline_issue; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.has_deadline_issue IS 'Is late';


--
-- Name: COLUMN stock_picking.printed; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.printed IS 'Printed';


--
-- Name: COLUMN stock_picking.is_locked; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.is_locked IS 'Is Locked';


--
-- Name: COLUMN stock_picking.scheduled_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.scheduled_date IS 'Scheduled Date';


--
-- Name: COLUMN stock_picking.date_deadline; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.date_deadline IS 'Deadline';


--
-- Name: COLUMN stock_picking.date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.date IS 'Creation Date';


--
-- Name: COLUMN stock_picking.date_done; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.date_done IS 'Date of Transfer';


--
-- Name: COLUMN stock_picking.create_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.create_date IS 'Created on';


--
-- Name: COLUMN stock_picking.write_date; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON COLUMN public.stock_picking.write_date IS 'Last Updated on';


--
-- Name: stock_picking_id_seq; Type: SEQUENCE; Schema: public; Owner: dimi
--

CREATE SEQUENCE public.stock_picking_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.stock_picking_id_seq OWNER TO dimi;

--
-- Name: stock_picking_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: dimi
--

ALTER SEQUENCE public.stock_picking_id_seq OWNED BY public.stock_picking.id;


--
-- Name: stock_quant_id_seq; Type: SEQUENCE; Schema: public; Owner: dimi
--

CREATE SEQUENCE public.stock_quant_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.stock_quant_id_seq OWNER TO dimi;

--
-- Name: stock_quant_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: dimi
--

ALTER SEQUENCE public.stock_quant_id_seq OWNED BY public.stock_quant.id;


--
-- Name: product_product id; Type: DEFAULT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.product_product ALTER COLUMN id SET DEFAULT nextval('public.product_product_id_seq'::regclass);


--
-- Name: stock_location id; Type: DEFAULT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location ALTER COLUMN id SET DEFAULT nextval('public.stock_location_id_seq'::regclass);


--
-- Name: stock_move_line id; Type: DEFAULT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line ALTER COLUMN id SET DEFAULT nextval('public.stock_move_line_id_seq'::regclass);


--
-- Name: stock_picking id; Type: DEFAULT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking ALTER COLUMN id SET DEFAULT nextval('public.stock_picking_id_seq'::regclass);


--
-- Name: stock_quant id; Type: DEFAULT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant ALTER COLUMN id SET DEFAULT nextval('public.stock_quant_id_seq'::regclass);


--
-- Name: product_product product_product_pkey; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.product_product
    ADD CONSTRAINT product_product_pkey PRIMARY KEY (id);


--
-- Name: stock_location stock_location_barcode_company_uniq; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_barcode_company_uniq UNIQUE (barcode, company_id);


--
-- Name: CONSTRAINT stock_location_barcode_company_uniq ON stock_location; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON CONSTRAINT stock_location_barcode_company_uniq ON public.stock_location IS 'unique (barcode,company_id)';


--
-- Name: stock_location stock_location_pkey; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_pkey PRIMARY KEY (id);


--
-- Name: stock_move_line stock_move_line_pkey; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_pkey PRIMARY KEY (id);


--
-- Name: stock_picking stock_picking_name_uniq; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_name_uniq UNIQUE (name, company_id);


--
-- Name: CONSTRAINT stock_picking_name_uniq ON stock_picking; Type: COMMENT; Schema: public; Owner: dimi
--

COMMENT ON CONSTRAINT stock_picking_name_uniq ON public.stock_picking IS 'unique(name, company_id)';


--
-- Name: stock_picking stock_picking_pkey; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_pkey PRIMARY KEY (id);


--
-- Name: stock_quant stock_quant_pkey; Type: CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_pkey PRIMARY KEY (id);


--
-- Name: product_product__barcode_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX product_product__barcode_index ON public.product_product USING btree (barcode) WHERE (barcode IS NOT NULL);


--
-- Name: product_product__combination_indices_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX product_product__combination_indices_index ON public.product_product USING btree (combination_indices);


--
-- Name: product_product__default_code_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX product_product__default_code_index ON public.product_product USING btree (default_code);


--
-- Name: product_product__product_tmpl_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX product_product__product_tmpl_id_index ON public.product_product USING btree (product_tmpl_id);


--
-- Name: product_product_combination_unique; Type: INDEX; Schema: public; Owner: dimi
--

CREATE UNIQUE INDEX product_product_combination_unique ON public.product_product USING btree (product_tmpl_id, combination_indices) WHERE (active IS TRUE);


--
-- Name: stock_location__company_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_location__company_id_index ON public.stock_location USING btree (company_id);


--
-- Name: stock_location__location_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_location__location_id_index ON public.stock_location USING btree (location_id);


--
-- Name: stock_location__parent_path_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_location__parent_path_index ON public.stock_location USING btree (parent_path);


--
-- Name: stock_location__usage_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_location__usage_index ON public.stock_location USING btree (usage);


--
-- Name: stock_move_line__company_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_move_line__company_id_index ON public.stock_move_line USING btree (company_id);


--
-- Name: stock_move_line__move_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_move_line__move_id_index ON public.stock_move_line USING btree (move_id);


--
-- Name: stock_move_line__owner_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_move_line__owner_id_index ON public.stock_move_line USING btree (owner_id) WHERE (owner_id IS NOT NULL);


--
-- Name: stock_move_line__picking_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_move_line__picking_id_index ON public.stock_move_line USING btree (picking_id);


--
-- Name: stock_move_line__product_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_move_line__product_id_index ON public.stock_move_line USING btree (product_id);


--
-- Name: stock_move_line_free_reservation_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_move_line_free_reservation_index ON public.stock_move_line USING btree (id, company_id, product_id, lot_id, location_id, owner_id, package_id) WHERE (((state IS NULL) OR ((state)::text <> ALL ((ARRAY['cancel'::character varying, 'done'::character varying])::text[]))) AND (quantity_product_uom > (0)::numeric) AND (NOT picked));


--
-- Name: stock_picking__backorder_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__backorder_id_index ON public.stock_picking USING btree (backorder_id) WHERE (backorder_id IS NOT NULL);


--
-- Name: stock_picking__company_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__company_id_index ON public.stock_picking USING btree (company_id);


--
-- Name: stock_picking__owner_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__owner_id_index ON public.stock_picking USING btree (owner_id) WHERE (owner_id IS NOT NULL);


--
-- Name: stock_picking__partner_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__partner_id_index ON public.stock_picking USING btree (partner_id) WHERE (partner_id IS NOT NULL);


--
-- Name: stock_picking__picking_type_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__picking_type_id_index ON public.stock_picking USING btree (picking_type_id);


--
-- Name: stock_picking__return_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__return_id_index ON public.stock_picking USING btree (return_id) WHERE (return_id IS NOT NULL);


--
-- Name: stock_picking__scheduled_date_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__scheduled_date_index ON public.stock_picking USING btree (scheduled_date);


--
-- Name: stock_picking__state_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_picking__state_index ON public.stock_picking USING btree (state);


--
-- Name: stock_quant__location_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_quant__location_id_index ON public.stock_quant USING btree (location_id);


--
-- Name: stock_quant__lot_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_quant__lot_id_index ON public.stock_quant USING btree (lot_id);


--
-- Name: stock_quant__owner_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_quant__owner_id_index ON public.stock_quant USING btree (owner_id) WHERE (owner_id IS NOT NULL);


--
-- Name: stock_quant__package_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_quant__package_id_index ON public.stock_quant USING btree (package_id);


--
-- Name: stock_quant__product_id_index; Type: INDEX; Schema: public; Owner: dimi
--

CREATE INDEX stock_quant__product_id_index ON public.stock_quant USING btree (product_id);


--
-- Name: product_product product_product_create_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.product_product
    ADD CONSTRAINT product_product_create_uid_fkey FOREIGN KEY (create_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: product_product product_product_product_tmpl_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.product_product
    ADD CONSTRAINT product_product_product_tmpl_id_fkey FOREIGN KEY (product_tmpl_id) REFERENCES public.product_template(id) ON DELETE CASCADE;


--
-- Name: product_product product_product_write_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.product_product
    ADD CONSTRAINT product_product_write_uid_fkey FOREIGN KEY (write_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_location stock_location_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_company(id) ON DELETE SET NULL;


--
-- Name: stock_location stock_location_create_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_create_uid_fkey FOREIGN KEY (create_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_location stock_location_location_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_location(id) ON DELETE CASCADE;


--
-- Name: stock_location stock_location_removal_strategy_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_removal_strategy_id_fkey FOREIGN KEY (removal_strategy_id) REFERENCES public.product_removal(id) ON DELETE SET NULL;


--
-- Name: stock_location stock_location_storage_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_storage_category_id_fkey FOREIGN KEY (storage_category_id) REFERENCES public.stock_storage_category(id) ON DELETE SET NULL;


--
-- Name: stock_location stock_location_warehouse_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.stock_warehouse(id) ON DELETE SET NULL;


--
-- Name: stock_location stock_location_write_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_location
    ADD CONSTRAINT stock_location_write_uid_fkey FOREIGN KEY (write_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_company(id) ON DELETE RESTRICT;


--
-- Name: stock_move_line stock_move_line_create_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_create_uid_fkey FOREIGN KEY (create_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_location_dest_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_location(id) ON DELETE RESTRICT;


--
-- Name: stock_move_line stock_move_line_location_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_location(id) ON DELETE RESTRICT;


--
-- Name: stock_move_line stock_move_line_lot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_lot_id_fkey FOREIGN KEY (lot_id) REFERENCES public.stock_lot(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_move_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.stock_move(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.res_partner(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_package_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_package_id_fkey FOREIGN KEY (package_id) REFERENCES public.stock_quant_package(id) ON DELETE RESTRICT;


--
-- Name: stock_move_line stock_move_line_package_level_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_package_level_id_fkey FOREIGN KEY (package_level_id) REFERENCES public.stock_package_level(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_picking_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_picking_id_fkey FOREIGN KEY (picking_id) REFERENCES public.stock_picking(id) ON DELETE SET NULL;


--
-- Name: stock_move_line stock_move_line_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_product(id) ON DELETE CASCADE;


--
-- Name: stock_move_line stock_move_line_product_uom_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_product_uom_id_fkey FOREIGN KEY (product_uom_id) REFERENCES public.uom_uom(id) ON DELETE RESTRICT;


--
-- Name: stock_move_line stock_move_line_result_package_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_result_package_id_fkey FOREIGN KEY (result_package_id) REFERENCES public.stock_quant_package(id) ON DELETE RESTRICT;


--
-- Name: stock_move_line stock_move_line_write_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_move_line
    ADD CONSTRAINT stock_move_line_write_uid_fkey FOREIGN KEY (write_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_backorder_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_backorder_id_fkey FOREIGN KEY (backorder_id) REFERENCES public.stock_picking(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_company(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_create_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_create_uid_fkey FOREIGN KEY (create_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.procurement_group(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_location_dest_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_location(id) ON DELETE RESTRICT;


--
-- Name: stock_picking stock_picking_location_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_location(id) ON DELETE RESTRICT;


--
-- Name: stock_picking stock_picking_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.res_partner(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_partner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partner(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_picking_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_picking_type_id_fkey FOREIGN KEY (picking_type_id) REFERENCES public.stock_picking_type(id) ON DELETE RESTRICT;


--
-- Name: stock_picking stock_picking_return_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_return_id_fkey FOREIGN KEY (return_id) REFERENCES public.stock_picking(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_picking stock_picking_write_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_picking
    ADD CONSTRAINT stock_picking_write_uid_fkey FOREIGN KEY (write_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_quant stock_quant_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_company(id) ON DELETE SET NULL;


--
-- Name: stock_quant stock_quant_create_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_create_uid_fkey FOREIGN KEY (create_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_quant stock_quant_location_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_location(id) ON DELETE RESTRICT;


--
-- Name: stock_quant stock_quant_lot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_lot_id_fkey FOREIGN KEY (lot_id) REFERENCES public.stock_lot(id) ON DELETE RESTRICT;


--
-- Name: stock_quant stock_quant_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.res_partner(id) ON DELETE SET NULL;


--
-- Name: stock_quant stock_quant_package_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_package_id_fkey FOREIGN KEY (package_id) REFERENCES public.stock_quant_package(id) ON DELETE RESTRICT;


--
-- Name: stock_quant stock_quant_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_product(id) ON DELETE RESTRICT;


--
-- Name: stock_quant stock_quant_storage_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_storage_category_id_fkey FOREIGN KEY (storage_category_id) REFERENCES public.stock_storage_category(id) ON DELETE SET NULL;


--
-- Name: stock_quant stock_quant_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- Name: stock_quant stock_quant_write_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: dimi
--

ALTER TABLE ONLY public.stock_quant
    ADD CONSTRAINT stock_quant_write_uid_fkey FOREIGN KEY (write_uid) REFERENCES public.res_users(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--

\unrestrict bCjjil46YuV2CH1pXFVzyU8a5hjJit6d4xaxHPzLw45Acug6Uw1ficJDKghSA8a

