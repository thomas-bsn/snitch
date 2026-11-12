from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.notification_webhook import NotificationWebhook
from ...models.notification_webhook_input import NotificationWebhookInput
from typing import cast



def _get_kwargs(
    *,
    body: NotificationWebhookInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/notification-webhook",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | NotificationWebhook:
    if response.status_code == 200:
        response_200 = NotificationWebhook.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | NotificationWebhook]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: NotificationWebhookInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | NotificationWebhook]:
    """  Réservé au Super-administrateur. Enregistre les réglages ; url absente conserve l'adresse
    enregistrée, une nouvelle la remplace (rotation). Seule une adresse https d'exécution de webhook
    Discord est acceptée (400 invalid_discord_webhook) ; activer sans adresse renvoie 400
    discord_webhook_required ; catégorie inconnue 400 invalid_webhook_category. Les livraisons en
    attente que les nouveaux réglages n'autorisent plus sont abandonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (NotificationWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | NotificationWebhook]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body: NotificationWebhookInput,
    origin: str,
    x_csrf_token: str,

) -> Error | NotificationWebhook | None:
    """  Réservé au Super-administrateur. Enregistre les réglages ; url absente conserve l'adresse
    enregistrée, une nouvelle la remplace (rotation). Seule une adresse https d'exécution de webhook
    Discord est acceptée (400 invalid_discord_webhook) ; activer sans adresse renvoie 400
    discord_webhook_required ; catégorie inconnue 400 invalid_webhook_category. Les livraisons en
    attente que les nouveaux réglages n'autorisent plus sont abandonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (NotificationWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | NotificationWebhook
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: NotificationWebhookInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | NotificationWebhook]:
    """  Réservé au Super-administrateur. Enregistre les réglages ; url absente conserve l'adresse
    enregistrée, une nouvelle la remplace (rotation). Seule une adresse https d'exécution de webhook
    Discord est acceptée (400 invalid_discord_webhook) ; activer sans adresse renvoie 400
    discord_webhook_required ; catégorie inconnue 400 invalid_webhook_category. Les livraisons en
    attente que les nouveaux réglages n'autorisent plus sont abandonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (NotificationWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | NotificationWebhook]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: NotificationWebhookInput,
    origin: str,
    x_csrf_token: str,

) -> Error | NotificationWebhook | None:
    """  Réservé au Super-administrateur. Enregistre les réglages ; url absente conserve l'adresse
    enregistrée, une nouvelle la remplace (rotation). Seule une adresse https d'exécution de webhook
    Discord est acceptée (400 invalid_discord_webhook) ; activer sans adresse renvoie 400
    discord_webhook_required ; catégorie inconnue 400 invalid_webhook_category. Les livraisons en
    attente que les nouveaux réglages n'autorisent plus sont abandonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (NotificationWebhookInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | NotificationWebhook
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
