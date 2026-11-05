from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: str,
    *,
    worker_id: str,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token





    params: dict[str, Any] = {}

    params["worker_id"] = worker_id


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/admin/voice-models/{id}/installation".format(id=quote(str(id), safe=""),),
        "params": params,
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
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
    worker_id: str,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Administrateurs uniquement. Ordonne à UN Worker, désigné par worker_id (400 worker_required sinon),
    de supprimer un modèle installé. Refusé (voice_model_in_use) s'il est actif ou candidat, ou utilisé
    par un job de ce Worker ; 409 voice_model_not_removable si ce Worker ne le détient pas.

    Args:
        id (str):
        worker_id (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        id=id,
worker_id=worker_id,
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
    worker_id: str,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Administrateurs uniquement. Ordonne à UN Worker, désigné par worker_id (400 worker_required sinon),
    de supprimer un modèle installé. Refusé (voice_model_in_use) s'il est actif ou candidat, ou utilisé
    par un job de ce Worker ; 409 voice_model_not_removable si ce Worker ne le détient pas.

    Args:
        id (str):
        worker_id (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        id=id,
client=client,
worker_id=worker_id,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    worker_id: str,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Administrateurs uniquement. Ordonne à UN Worker, désigné par worker_id (400 worker_required sinon),
    de supprimer un modèle installé. Refusé (voice_model_in_use) s'il est actif ou candidat, ou utilisé
    par un job de ce Worker ; 409 voice_model_not_removable si ce Worker ne le détient pas.

    Args:
        id (str):
        worker_id (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        id=id,
worker_id=worker_id,
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
    worker_id: str,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Administrateurs uniquement. Ordonne à UN Worker, désigné par worker_id (400 worker_required sinon),
    de supprimer un modèle installé. Refusé (voice_model_in_use) s'il est actif ou candidat, ou utilisé
    par un job de ce Worker ; 409 voice_model_not_removable si ce Worker ne le détient pas.

    Args:
        id (str):
        worker_id (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
worker_id=worker_id,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
