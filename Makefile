SHELL := /bin/zsh

ROOT_DIR := $(CURDIR)
FRONTEND_DIR := $(ROOT_DIR)/frontend

.PHONY: help deps deps-root deps-frontend frontend-build frontend-dev backend-build backend-run testserver-run web-run web-dev desktop-build desktop-dev desktop-open desktop-start desktop-dist clean

help:
	@printf '%s\n' \
		'deps           Install root and frontend npm dependencies' \
		'deps-root      Install root Electron dependencies' \
		'deps-frontend  Install frontend dependencies' \
		'frontend-build Build Vue app into static/' \
		'frontend-dev   Run Vite dev server only' \
		'backend-build  Build Go backend binary into dist/' \
		'desktop-build  Build frontend and backend for Electron' \
		'backend-run    Run Go backend directly' \
		'testserver-run Run gRPC test server from ./testserver' \
		'web-run        Build frontend and run Go backend' \
		'web-dev        Run backend and Vite together for browser development' \
		'desktop-dev    Run Electron + Vite + auto-started backend' \
		'desktop-open   Open Electron against built backend/static assets' \
		'desktop-start  Build everything and open Electron' \
		'desktop-dist   Build packaged Electron app into release/' \
		'clean          Remove generated desktop/web build artifacts'

deps: deps-root deps-frontend

deps-root:
	npm ci

deps-frontend:
	npm --prefix $(FRONTEND_DIR) ci

frontend-build:
	npm --prefix $(FRONTEND_DIR) run build

frontend-dev:
	npm --prefix $(FRONTEND_DIR) run dev

backend-build:
	npm run build:backend

desktop-build:
	npm run desktop:build

backend-run:
	cd backend && go run .

testserver-run:
	cd backend && go run ./testserver

web-run:
	npm run build:frontend
	cd backend && go run .

web-dev:
	trap 'kill 0' EXIT INT TERM; \
	cd backend && GRPC_EXPLORER_NO_BROWSER=1 GRPC_EXPLORER_PORT=22333 go run . & \
	npm --prefix $(FRONTEND_DIR) run dev

desktop-dev:
	npm run desktop:dev

desktop-open:
	npm run desktop:open

desktop-start:
	npm run desktop:start

desktop-dist:
	npm run desktop:dist

clean:
	rm -rf .cache dist release static frontend/dist frontend/tsconfig.tsbuildinfo
