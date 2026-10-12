# ws-client can be used to execute commands:
# set a live sessionID to config.json to loginKey
# run npm rund dev, it connects to localhost server, so
# run it first.

curl -v -i \
	-X POST 'http://localhost:59000/session/bootstrap' \
	-H 'Content-Type: application/json'
32d84e30-0d55-8686-dd6c-d3114e0fea1b
1a1c715a-fa30-7043-ac16-eeca43806da4

#real address:https://eurobeton.katren.org:59902
#-X POST 'http://localhost:59000/api-post/elkon-production-cmd/fetch-config' \
#-X POST 'https://eurobeton.katren.org:59902/api-post/elkon-production-cmd/fetch-config' \ 
#-X POST 'https://192.168.1.177:59902/api-post/elkon-production-cmd/fetch-config' \ 
#this works fine for localhost, session is the eurtobeton active session

curl -v -i \
	'http://localhost:59000/api-post/elkon-production-cmd/fetch-config' \
	-b '_s=00c15922-b8e6-7c90-32f6-33ce229699a3' \
	-H 'Content-Type: application/json' \
	-H 'Query-Id: 111222555' \
	-d '{"production_site_id":2}'

curl -v -i \
	'https://eurobeton.katren.org:59904/api-post/elkon-production-cmd/fetch-config' \
	-b '_s=ij1ge9p58g2a6q85qhpmqo6f7i' \
	-H 'Content-Type: application/json' \
	-H 'Query-Id: 111222555' \
	-d '{"production_site_id":2}'

curl 'http://178.46.157.185:59900/constant/list' -b '_s=ij1ge9p58g2a6q85qhpmqo6f7i'

curl -v -i \
	-X POST 'http://localhost:59000/api-post/elkon-production-cmd/fetch-production' \
	-b '_s=ij1ge9p58g2a6q85qhpmqo6f7i' \
	-H 'Content-Type: application/json' \
	-H 'Query-Id: 111222444' \
	-d '{"production_site_id":4, "production_id": 20695}'

curl http://localhost:59000/?c=User_Controller&f=login_ks&k=01d85f3f44459c90228be81305dd59eb&v=ViewJSON

SELECT elkon_session_id,name
FROM production_sites
WHERE id = 4

select * from distrib_commands 

select pg_notify('Distrib.Command', 
	json_build_object(
		'recipient_sess_id','', 
		'payload', json_build_object(
			'cmd', 'fetch_config', 
			'query_id', '111222555'
		)
	)::text)
