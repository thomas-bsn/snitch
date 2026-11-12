from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.notification_webhook import NotificationWebhook
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/notification-webhook",
    }


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

) -> Response[Error | NotificationWebhook]:
    """  Administrateurs. Réglages du webhook Discord des notifications (#283, ADR 0067) : l'adresse n'est
    jamais renvoyée (configured et hint seulement), avec le statut de la dernière livraison, les
    livraisons récentes et le journal des changements.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | NotificationWebhook]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,

) -> Error | NotificationWebhook | None:
    """  Administrateurs. Réglages du webhook Discord des notifications (#283, ADR 0067) : l'adresse n'est
    jamais renvoyée (configured et hint seulement), avec le statut de la dernière livraison, les
    livraisons récentes et le journal des changements.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | NotificationWebhook
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | NotificationWebhook]:
    """  Administrateurs. Réglages du webhook Discord des notifications (#283, ADR 0067) : l'adresse n'est
    jamais renvoyée (configured et hint seulement), avec le statut de la dernière livraison, les
    livraisons récentes et le journal des changements.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | NotificationWebhook]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,

) -> Error | NotificationWebhook | None:
    """  Administrateurs. Réglages du webhook Discord des notifications (#283, ADR 0067) : l'adresse n'est
    jamais renvoyée (configured et hint seulement), avec le statut de la dernière livraison, les
    livraisons récentes et le journal des changements.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | NotificationWebhook
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
