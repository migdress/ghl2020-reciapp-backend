.PHONY: deploy_login
deploy_login: 
	make -C login deploy


.PHONY: deploy_get_picking_routes
deploy_get_picking_routes: 
	make -C get_picking_routes deploy
