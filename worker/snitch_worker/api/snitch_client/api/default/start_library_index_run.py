from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.index_run_answer import IndexRunAnswer
from ...models.start_library_index_run_body import StartLibraryIndexRunBody
from typing import cast



def _get_kwargs(
    *,
    body: StartLibraryIndexRunBody,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/library/index-runs",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | IndexRunAnswer:
    if response.status_code == 201:
        response_201 = IndexRunAnswer.from_dict(response.json())



        return response_201

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())



        return response_400

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    if response.status_code == 422:
        response_422 = Error.from_dict(response.json())



        return response_422

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | IndexRunAnswer]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: StartLibraryIndexRunBody,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | IndexRunAnswer]:
    """ Indexer toutes les vidéos de la Racine qui n'ont pas encore de fiche clip (administrateur, action
    explicite)

     mode cards : fiches seulement. mode full : fiches puis analyse complète de chaque nouveau clip, même
    si l'analyse automatique est désactivée. Une vidéo qui a déjà un clip n'est jamais retraitée. Une
    seule indexation en cours à la fois. Le travail est fait par le scan géré existant : il survit à un
    redémarrage.

    Args:
        origin (str):
        x_csrf_token (str):
        body (StartLibraryIndexRunBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | IndexRunAnswer]
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
    client: AuthenticatedClient,
    body: StartLibraryIndexRunBody,
    origin: str,
    x_csrf_token: str,

) -> Error | IndexRunAnswer | None:
    """ Indexer toutes les vidéos de la Racine qui n'ont pas encore de fiche clip (administrateur, action
    explicite)

     mode cards : fiches seulement. mode full : fiches puis analyse complète de chaque nouveau clip, même
    si l'analyse automatique est désactivée. Une vidéo qui a déjà un clip n'est jamais retraitée. Une
    seule indexation en cours à la fois. Le travail est fait par le scan géré existant : il survit à un
    redémarrage.

    Args:
        origin (str):
        x_csrf_token (str):
        body (StartLibraryIndexRunBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | IndexRunAnswer
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: StartLibraryIndexRunBody,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | IndexRunAnswer]:
    """ Indexer toutes les vidéos de la Racine qui n'ont pas encore de fiche clip (administrateur, action
    explicite)

     mode cards : fiches seulement. mode full : fiches puis analyse complète de chaque nouveau clip, même
    si l'analyse automatique est désactivée. Une vidéo qui a déjà un clip n'est jamais retraitée. Une
    seule indexation en cours à la fois. Le travail est fait par le scan géré existant : il survit à un
    redémarrage.

    Args:
        origin (str):
        x_csrf_token (str):
        body (StartLibraryIndexRunBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | IndexRunAnswer]
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
    client: AuthenticatedClient,
    body: StartLibraryIndexRunBody,
    origin: str,
    x_csrf_token: str,

) -> Error | IndexRunAnswer | None:
    """ Indexer toutes les vidéos de la Racine qui n'ont pas encore de fiche clip (administrateur, action
    explicite)

     mode cards : fiches seulement. mode full : fiches puis analyse complète de chaque nouveau clip, même
    si l'analyse automatique est désactivée. Une vidéo qui a déjà un clip n'est jamais retraitée. Une
    seule indexation en cours à la fois. Le travail est fait par le scan géré existant : il survit à un
    redémarrage.

    Args:
        origin (str):
        x_csrf_token (str):
        body (StartLibraryIndexRunBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | IndexRunAnswer
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
