from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model_order_history import VoiceModelOrderHistory
from typing import cast



def _get_kwargs(
    id: str,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/workers/{id}/voice-model-orders".format(id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModelOrderHistory:
    if response.status_code == 200:
        response_200 = VoiceModelOrderHistory.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModelOrderHistory]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceModelOrderHistory]:
    """  Administrateurs uniquement. Historique des ordres de modèle d'un Worker (installation, retrait), du
    plus récent au plus ancien, avec leur issue, leur cause et leur auteur.

    Args:
        id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelOrderHistory]
     """


    kwargs = _get_kwargs(
        id=id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient,

) -> Error | VoiceModelOrderHistory | None:
    """  Administrateurs uniquement. Historique des ordres de modèle d'un Worker (installation, retrait), du
    plus récent au plus ancien, avec leur issue, leur cause et leur auteur.

    Args:
        id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelOrderHistory
     """


    return sync_detailed(
        id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceModelOrderHistory]:
    """  Administrateurs uniquement. Historique des ordres de modèle d'un Worker (installation, retrait), du
    plus récent au plus ancien, avec leur issue, leur cause et leur auteur.

    Args:
        id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelOrderHistory]
     """


    kwargs = _get_kwargs(
        id=id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient,

) -> Error | VoiceModelOrderHistory | None:
    """  Administrateurs uniquement. Historique des ordres de modèle d'un Worker (installation, retrait), du
    plus récent au plus ancien, avec leur issue, leur cause et leur auteur.

    Args:
        id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelOrderHistory
     """


    return (await asyncio_detailed(
        id=id,
client=client,

    )).parsed
