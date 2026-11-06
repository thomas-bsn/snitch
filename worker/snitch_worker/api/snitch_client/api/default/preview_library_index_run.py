from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.index_run_preview import IndexRunPreview
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/library/index-runs/preview",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | IndexRunPreview:
    if response.status_code == 200:
        response_200 = IndexRunPreview.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    if response.status_code == 422:
        response_422 = Error.from_dict(response.json())



        return response_422

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | IndexRunPreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | IndexRunPreview]:
    """ Estimer l'indexation de toute la Racine vidéo, sans la démarrer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos). Ne compte ni les dossiers cachés,
    ni les liens symboliques, ni les fichiers non vidéo ; les Dossiers privés ne sont comptés que pour
    un Super-administrateur.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | IndexRunPreview]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | IndexRunPreview | None:
    """ Estimer l'indexation de toute la Racine vidéo, sans la démarrer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos). Ne compte ni les dossiers cachés,
    ni les liens symboliques, ni les fichiers non vidéo ; les Dossiers privés ne sont comptés que pour
    un Super-administrateur.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | IndexRunPreview
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | IndexRunPreview]:
    """ Estimer l'indexation de toute la Racine vidéo, sans la démarrer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos). Ne compte ni les dossiers cachés,
    ni les liens symboliques, ni les fichiers non vidéo ; les Dossiers privés ne sont comptés que pour
    un Super-administrateur.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | IndexRunPreview]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | IndexRunPreview | None:
    """ Estimer l'indexation de toute la Racine vidéo, sans la démarrer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos). Ne compte ni les dossiers cachés,
    ni les liens symboliques, ni les fichiers non vidéo ; les Dossiers privés ne sont comptés que pour
    un Super-administrateur.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | IndexRunPreview
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
