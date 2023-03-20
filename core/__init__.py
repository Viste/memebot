from aiogram import Router


def setup_routers() -> Router:
    from . import core

    router = Router()

    router.include_router(core.router)

    return router
