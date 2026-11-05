from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model_order_input import VoiceModelOrderInput
from ...models.worker_model_order import WorkerModelOrder
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: VoiceModelOrderInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/workers/{id}/voice-model-orders".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerModelOrder:
    if response.status_code == 200:
        response_200 = WorkerModelOrder.from_dict(response.json())



        return response_200

    if response.status_code == 201:
        response_201 = WorkerModelOrder.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerModelOrder]:
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
    body: VoiceModelOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerModelOrder]:
    """  Administrateurs uniquement. Ordonne à un Worker d'installer un modèle du catalogue. Rejouer la même
    clé d'idempotence pour le même Worker et le même modèle renvoie l'ordre déjà enregistré (200) sans
    effet double ; la même clé pour autre chose est refusée (idempotency_key_reused). Refusé si le
    modèle est inconnu, désactivé ou non pris en charge, déjà installé, déjà demandé, ou si le type est
    occupé par un autre modèle.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerModelOrder]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerModelOrder | None:
    """  Administrateurs uniquement. Ordonne à un Worker d'installer un modèle du catalogue. Rejouer la même
    clé d'idempotence pour le même Worker et le même modèle renvoie l'ordre déjà enregistré (200) sans
    effet double ; la même clé pour autre chose est refusée (idempotency_key_reused). Refusé si le
    modèle est inconnu, désactivé ou non pris en charge, déjà installé, déjà demandé, ou si le type est
    occupé par un autre modèle.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerModelOrder
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerModelOrder]:
    """  Administrateurs uniquement. Ordonne à un Worker d'installer un modèle du catalogue. Rejouer la même
    clé d'idempotence pour le même Worker et le même modèle renvoie l'ordre déjà enregistré (200) sans
    effet double ; la même clé pour autre chose est refusée (idempotency_key_reused). Refusé si le
    modèle est inconnu, désactivé ou non pris en charge, déjà installé, déjà demandé, ou si le type est
    occupé par un autre modèle.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerModelOrder]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerModelOrder | None:
    """  Administrateurs uniquement. Ordonne à un Worker d'installer un modèle du catalogue. Rejouer la même
    clé d'idempotence pour le même Worker et le même modèle renvoie l'ordre déjà enregistré (200) sans
    effet double ; la même clé pour autre chose est refusée (idempotency_key_reused). Refusé si le
    modèle est inconnu, désactivé ou non pris en charge, déjà installé, déjà demandé, ou si le type est
    occupé par un autre modèle.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerModelOrder
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
