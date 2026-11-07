from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.folder_logo_state import FolderLogoState
from ...types import File, FileTypes
from io import BytesIO
from typing import cast



def _get_kwargs(
    *,
    body:    File  |     File  | Unset = UNSET,
    path: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}




    params: dict[str, Any] = {}

    params["path"] = path


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/library/folder-logo",
        "params": params,
    }

    if isinstance(body, File):
        _kwargs["content"] = body.payload
        headers["Content-Type"] = "image/png"
    if isinstance(body, File):
        _kwargs["content"] = body.payload
        headers["Content-Type"] = "image/jpeg"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | FolderLogoState:
    if response.status_code == 200:
        response_200 = FolderLogoState.from_dict(response.json())



        return response_200

    if response.status_code == 413:
        response_413 = Error.from_dict(response.json())



        return response_413

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | FolderLogoState]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    path: str,

) -> Response[Error | FolderLogoState]:
    """ Choisir un logo personnalisé pour un dossier (administrateur ; Super-administrateur pour un Dossier
    privé)

     Le corps est l'image elle-même (PNG ou JPEG, 2 Mo maximum). La nouvelle image devient le logo
    courant et entre dans l'historique de quatre versions.

    Args:
        path (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FolderLogoState]
     """


    kwargs = _get_kwargs(
        body=body,
path=path,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    path: str,

) -> Error | FolderLogoState | None:
    """ Choisir un logo personnalisé pour un dossier (administrateur ; Super-administrateur pour un Dossier
    privé)

     Le corps est l'image elle-même (PNG ou JPEG, 2 Mo maximum). La nouvelle image devient le logo
    courant et entre dans l'historique de quatre versions.

    Args:
        path (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FolderLogoState
     """


    return sync_detailed(
        client=client,
body=body,
path=path,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    path: str,

) -> Response[Error | FolderLogoState]:
    """ Choisir un logo personnalisé pour un dossier (administrateur ; Super-administrateur pour un Dossier
    privé)

     Le corps est l'image elle-même (PNG ou JPEG, 2 Mo maximum). La nouvelle image devient le logo
    courant et entre dans l'historique de quatre versions.

    Args:
        path (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FolderLogoState]
     """


    kwargs = _get_kwargs(
        body=body,
path=path,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    path: str,

) -> Error | FolderLogoState | None:
    """ Choisir un logo personnalisé pour un dossier (administrateur ; Super-administrateur pour un Dossier
    privé)

     Le corps est l'image elle-même (PNG ou JPEG, 2 Mo maximum). La nouvelle image devient le logo
    courant et entre dans l'historique de quatre versions.

    Args:
        path (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FolderLogoState
     """


    return (await asyncio_detailed(
        client=client,
body=body,
path=path,

    )).parsed
