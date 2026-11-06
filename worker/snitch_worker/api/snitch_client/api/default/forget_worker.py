from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: str,
    *,
    force: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token





    params: dict[str, Any] = {}

    params["force"] = force


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/admin/workers/{id}/record".format(id=quote(str(id), safe=""),),
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
    client: AuthenticatedClient | Client,
    force: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Retire de la liste un Worker révoqué ; les jobs et événements gardent leur historique. 409
    worker_not_revoked tant qu'il n'est pas révoqué, worker_cleanup_pending tant que la machine n'a pas
    confirmé la révocation, sauf si force=true (choix explicite de l'administrateur : le nettoyage local
    n'est alors pas confirmé et le volume de modèles, le conteneur et le dossier du lanceur peuvent
    subsister sur la machine).

    Args:
        id (str):
        force (bool | Unset):  Default: False.
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
force=force,
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
    client: AuthenticatedClient | Client,
    force: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Retire de la liste un Worker révoqué ; les jobs et événements gardent leur historique. 409
    worker_not_revoked tant qu'il n'est pas révoqué, worker_cleanup_pending tant que la machine n'a pas
    confirmé la révocation, sauf si force=true (choix explicite de l'administrateur : le nettoyage local
    n'est alors pas confirmé et le volume de modèles, le conteneur et le dossier du lanceur peuvent
    subsister sur la machine).

    Args:
        id (str):
        force (bool | Unset):  Default: False.
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
force=force,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    force: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Retire de la liste un Worker révoqué ; les jobs et événements gardent leur historique. 409
    worker_not_revoked tant qu'il n'est pas révoqué, worker_cleanup_pending tant que la machine n'a pas
    confirmé la révocation, sauf si force=true (choix explicite de l'administrateur : le nettoyage local
    n'est alors pas confirmé et le volume de modèles, le conteneur et le dossier du lanceur peuvent
    subsister sur la machine).

    Args:
        id (str):
        force (bool | Unset):  Default: False.
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
force=force,
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
    client: AuthenticatedClient | Client,
    force: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Retire de la liste un Worker révoqué ; les jobs et événements gardent leur historique. 409
    worker_not_revoked tant qu'il n'est pas révoqué, worker_cleanup_pending tant que la machine n'a pas
    confirmé la révocation, sauf si force=true (choix explicite de l'administrateur : le nettoyage local
    n'est alors pas confirmé et le volume de modèles, le conteneur et le dossier du lanceur peuvent
    subsister sur la machine).

    Args:
        id (str):
        force (bool | Unset):  Default: False.
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
force=force,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
