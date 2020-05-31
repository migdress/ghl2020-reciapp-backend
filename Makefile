.PHONY: deploy_login
deploy_login: 
	make -C login deploy

.PHONY: deploy_get_open_shifts
deploy_get_open_shifts: 
	make -C get_open_shifts deploy

.PHONY: deploy_get_location_score
deploy_get_location_score: 
	make -C get_location_score deploy

.PHONY: deploy_pin_picking_point
deploy_pin_picking_point: 
	make -C pin_picking_point deploy

.PHONY: deploy_get_picking_routes
deploy_get_picking_routes: 
	make -C get_picking_routes deploy

.PHONY: deploy_get_assigned_routes
deploy_get_assigned_routes:
	make -C get_assigned_routes deploy

.PHONY: deploy_assign_picking_route
deploy_assign_picking_route:
	make -C assign_picking_route deploy

.PHONY: deploy_start_picking_route
deploy_start_picking_route: 
	make -C start_picking_route deploy

.PHONY: deploy_finish_picking_point
deploy_finish_picking_point: 
	make -C finish_picking_point deploy

.PHONY: deploy_all
deploy_all: 
	make -C assign_picking_route deploy
	make -C finish_picking_point deploy
	make -C get_assigned_routes deploy
	make -C get_location_score deploy
	make -C get_open_shifts deploy
	make -C get_picking_routes deploy
	make -C login deploy
	make -C pin_picking_point deploy
	make -C start_picking_route deploy



