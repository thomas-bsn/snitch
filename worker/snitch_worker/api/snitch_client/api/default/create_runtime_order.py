from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.runtime_order_input import RuntimeOrderInput
from ...models.worker_model_order import WorkerModelOrder
from typing import cast



def _get_kwargs(
    *,
    body: RuntimeOrderInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-model-runtime-orders",
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
    *,
    client: AuthenticatedClient | Client,
    body: RuntimeOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerModelOrder]:
    """  Administrateur : ordonne la mise à jour du runtime d'un modèle déjà prêt sur un Worker (update), ou
    le retour à une version déjà installée et approuvée (rollback). Les poids validés sont réutilisés
    sans téléchargement ; la liaison n'est publiée qu'après une inférence de test réussie, le modèle
    reste prêt sur l'ancienne liaison entre-temps. Aucun rollback vers une version révoquée. Causes de
    refus : voice_model_not_ready, runtime_order_in_progress, engine_release_unavailable,
    engine_release_mismatch, engine_already_bound, engine_not_installed, engine_order_in_progress,
    idempotency_key_reused, engine_not_found, invalid_runtime_order.

    Args:
        origin (str):
        x_csrf_token (str):
        body (RuntimeOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerModelOrder]
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
    body: RuntimeOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerModelOrder | None:
    """  Administrateur : ordonne la mise à jour du runtime d'un modèle déjà prêt sur un Worker (update), ou
    le retour à une version déjà installée et approuvée (rollback). Les poids validés sont réutilisés
    sans téléchargement ; la liaison n'est publiée qu'après une inférence de test réussie, le modèle
    reste prêt sur l'ancienne liaison entre-temps. Aucun rollback vers une version révoquée. Causes de
    refus : voice_model_not_ready, runtime_order_in_progress, engine_release_unavailable,
    engine_release_mismatch, engine_already_bound, engine_not_installed, engine_order_in_progress,
    idempotency_key_reused, engine_not_found, invalid_runtime_order.

    Args:
        origin (str):
        x_csrf_token (str):
        body (RuntimeOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerModelOrder
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
    body: RuntimeOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerModelOrder]:
    """  Administrateur : ordonne la mise à jour du runtime d'un modèle déjà prêt sur un Worker (update), ou
    le retour à une version déjà installée et approuvée (rollback). Les poids validés sont réutilisés
    sans téléchargement ; la liaison n'est publiée qu'après une inférence de test réussie, le modèle
    reste prêt sur l'ancienne liaison entre-temps. Aucun rollback vers une version révoquée. Causes de
    refus : voice_model_not_ready, runtime_order_in_progress, engine_release_unavailable,
    engine_release_mismatch, engine_already_bound, engine_not_installed, engine_order_in_progress,
    idempotency_key_reused, engine_not_found, invalid_runtime_order.

    Args:
        origin (str):
        x_csrf_token (str):
        body (RuntimeOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerModelOrder]
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
    body: RuntimeOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerModelOrder | None:
    """  Administrateur : ordonne la mise à jour du runtime d'un modèle déjà prêt sur un Worker (update), ou
    le retour à une version déjà installée et approuvée (rollback). Les poids validés sont réutilisés
    sans téléchargement ; la liaison n'est publiée qu'après une inférence de test réussie, le modèle
    reste prêt sur l'ancienne liaison entre-temps. Aucun rollback vers une version révoquée. Causes de
    refus : voice_model_not_ready, runtime_order_in_progress, engine_release_unavailable,
    engine_release_mismatch, engine_already_bound, engine_not_installed, engine_order_in_progress,
    idempotency_key_reused, engine_not_found, invalid_runtime_order.

    Args:
        origin (str):
        x_csrf_token (str):
        body (RuntimeOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerModelOrder
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
