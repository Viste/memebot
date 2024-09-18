import logging

from aiohttp import web

logger = logging.getLogger(__name__)


async def health_check(request):
    return web.Response(text="Bot is up", status=200)


async def start_web_app():
    app = web.Application()
    app.router.add_get("/healthz", health_check)
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", 8080)
    await site.start()
