from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.explorer_properties import ExplorerProperties
from typing import cast



def _get_kwargs(
    *,
    path: str,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["path"] = path


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/library/explorer/properties",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | ExplorerProperties:
    if response.status_code == 200:
        response_200 = ExplorerProperties.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | ExplorerProperties]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    path: str,

) -> Response[Error | ExplorerProperties]:
    """ Propriétés agrégées d'un dossier ou d'une vidéo de l'explorateur (ADR 0032)

     Un dossier agrège tout son sous-arbre : taille sur disque, nombre de vidéos et de sous-dossiers,
    durée et propriétaires des seuls clips lisibles, date du plus ancien ajout. Une vidéo donne sa
    durée, son propriétaire et ses dates ; clip:<id> répond pour un clip sans fichier dans la Racine.
    Les Dossiers privés et les clips non lisibles ne sont jamais comptés pour qui ne peut pas les
    ouvrir.

    Args:
        path (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ExplorerProperties]
     """


    kwargs = _get_kwargs(
        path=path,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    path: str,

) -> Error | ExplorerProperties | None:
    """ Propriétés agrégées d'un dossier ou d'une vidéo de l'explorateur (ADR 0032)

     Un dossier agrège tout son sous-arbre : taille sur disque, nombre de vidéos et de sous-dossiers,
    durée et propriétaires des seuls clips lisibles, date du plus ancien ajout. Une vidéo donne sa
    durée, son propriétaire et ses dates ; clip:<id> répond pour un clip sans fichier dans la Racine.
    Les Dossiers privés et les clips non lisibles ne sont jamais comptés pour qui ne peut pas les
    ouvrir.

    Args:
        path (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ExplorerProperties
     """


    return sync_detailed(
        client=client,
path=path,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    path: str,

) -> Response[Error | ExplorerProperties]:
    """ Propriétés agrégées d'un dossier ou d'une vidéo de l'explorateur (ADR 0032)

     Un dossier agrège tout son sous-arbre : taille sur disque, nombre de vidéos et de sous-dossiers,
    durée et propriétaires des seuls clips lisibles, date du plus ancien ajout. Une vidéo donne sa
    durée, son propriétaire et ses dates ; clip:<id> répond pour un clip sans fichier dans la Racine.
    Les Dossiers privés et les clips non lisibles ne sont jamais comptés pour qui ne peut pas les
    ouvrir.

    Args:
        path (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ExplorerProperties]
     """


    kwargs = _get_kwargs(
        path=path,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    path: str,

) -> Error | ExplorerProperties | None:
    """ Propriétés agrégées d'un dossier ou d'une vidéo de l'explorateur (ADR 0032)

     Un dossier agrège tout son sous-arbre : taille sur disque, nombre de vidéos et de sous-dossiers,
    durée et propriétaires des seuls clips lisibles, date du plus ancien ajout. Une vidéo donne sa
    durée, son propriétaire et ses dates ; clip:<id> répond pour un clip sans fichier dans la Racine.
    Les Dossiers privés et les clips non lisibles ne sont jamais comptés pour qui ne peut pas les
    ouvrir.

    Args:
        path (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ExplorerProperties
     """


    return (await asyncio_detailed(
        client=client,
path=path,

    )).parsed
