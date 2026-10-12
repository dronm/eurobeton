--select * from doc_material_procurements where number='000000004418'
--select * from ra_materials limit 1 
--select * from ra_cement where cement_silos_id = 4 order by date_time desc limit 10

 --786760 | 2025-10-31 14:34:06 | f   | material_fact_consumption | 3601527 |               4 |  2.599
--select * from material_fact_consumptions where id =  3601527 
--select * from productions where production_id = '51643'

/*
update material_fact_consumptions set
date_time = prod.production_dt_end
from (
select 
	p.production_id,
	p.production_dt_end,
	cons.date_time,
	cons.production_site_id
from material_fact_consumptions as cons
left join productions as p 
	on p.production_site_id = cons.production_site_id and p.production_id = cons.production_id
where cons.date_time is distinct from p.production_dt_end and p.production_dt_end is not null
	and p.production_dt_end::date >= '2025-10-30'
order by  p.production_dt_end desc
) as prod
where 
	prod.production_site_id = material_fact_consumptions.production_site_id 
	and prod.production_id = material_fact_consumptions.production_id
	*/

select 
	p.production_id,
	p.production_dt_end,
	cons.date_time,
	cons.production_site_id
from material_fact_consumptions as cons
left join productions as p 
	on p.production_site_id = cons.production_site_id and p.production_id = cons.production_id
where cons.date_time is distinct from p.production_dt_end and p.production_dt_end is not null
	and p.production_dt_end::date >= '2025-10-30'
order by  p.production_dt_end desc


