from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.explorer_listing import ExplorerListing
from ...models.list_library_explorer_recursive import ListLibraryExplorerRecursive
from ...models.list_library_explorer_state import ListLibraryExplorerState
from ...models.list_library_explorer_visibility import ListLibraryExplorerVisibility
from ...models.list_library_explorer_voices import ListLibraryExplorerVoices
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    *,
    path: str | Unset = UNSET,
    recursive: ListLibraryExplorerRecursive | Unset = UNSET,
    q: str | Unset = UNSET,
    state: ListLibraryExplorerState | Unset = UNSET,
    game: str | Unset = UNSET,
    visibility: ListLibraryExplorerVisibility | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    person_id: int | Unset = UNSET,
    voices: ListLibraryExplorerVoices | Unset = UNSET,
    limit: int | Unset = 100,
    offset: int | Unset = 0,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["path"] = path

    json_recursive: str | Unset = UNSET
    if not isinstance(recursive, Unset):
        json_recursive = recursive.value

    params["recursive"] = json_recursive

    params["q"] = q

    json_state: str | Unset = UNSET
    if not isinstance(state, Unset):
        json_state = state.value

    params["state"] = json_state

    params["game"] = game

    json_visibility: str | Unset = UNSET
    if not isinstance(visibility, Unset):
        json_visibility = visibility.value

    params["visibility"] = json_visibility

    json_from_: str | Unset = UNSET
    if not isinstance(from_, Unset):
        json_from_ = from_.isoformat()
    params["from"] = json_from_

    json_to: str | Unset = UNSET
    if not isinstance(to, Unset):
        json_to = to.isoformat()
    params["to"] = json_to

    params["owner_id"] = owner_id

    params["person_id"] = person_id

    json_voices: str | Unset = UNSET
    if not isinstance(voices, Unset):
        json_voices = voices.value

    params["voices"] = json_voices

    params["limit"] = limit

    params["offset"] = offset


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/library/explorer",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | ExplorerListing:
    if response.status_code == 200:
        response_200 = ExplorerListing.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | ExplorerListing]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,
    recursive: ListLibraryExplorerRecursive | Unset = UNSET,
    q: str | Unset = UNSET,
    state: ListLibraryExplorerState | Unset = UNSET,
    game: str | Unset = UNSET,
    visibility: ListLibraryExplorerVisibility | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    person_id: int | Unset = UNSET,
    voices: ListLibraryExplorerVoices | Unset = UNSET,
    limit: int | Unset = 100,
    offset: int | Unset = 0,

) -> Response[Error | ExplorerListing]:
    """ Explorateur de la Racine vidéo : dossiers et vidéos croisés avec la base, paginés côté serveur (ADR
    0032)

    Args:
        path (str | Unset):
        recursive (ListLibraryExplorerRecursive | Unset):
        q (str | Unset):
        state (ListLibraryExplorerState | Unset):
        game (str | Unset):
        visibility (ListLibraryExplorerVisibility | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):
        owner_id (int | Unset):
        person_id (int | Unset):
        voices (ListLibraryExplorerVoices | Unset):
        limit (int | Unset):  Default: 100.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ExplorerListing]
     """


    kwargs = _get_kwargs(
        path=path,
recursive=recursive,
q=q,
state=state,
game=game,
visibility=visibility,
from_=from_,
to=to,
owner_id=owner_id,
person_id=person_id,
voices=voices,
limit=limit,
offset=offset,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,
    recursive: ListLibraryExplorerRecursive | Unset = UNSET,
    q: str | Unset = UNSET,
    state: ListLibraryExplorerState | Unset = UNSET,
    game: str | Unset = UNSET,
    visibility: ListLibraryExplorerVisibility | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    person_id: int | Unset = UNSET,
    voices: ListLibraryExplorerVoices | Unset = UNSET,
    limit: int | Unset = 100,
    offset: int | Unset = 0,

) -> Error | ExplorerListing | None:
    """ Explorateur de la Racine vidéo : dossiers et vidéos croisés avec la base, paginés côté serveur (ADR
    0032)

    Args:
        path (str | Unset):
        recursive (ListLibraryExplorerRecursive | Unset):
        q (str | Unset):
        state (ListLibraryExplorerState | Unset):
        game (str | Unset):
        visibility (ListLibraryExplorerVisibility | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):
        owner_id (int | Unset):
        person_id (int | Unset):
        voices (ListLibraryExplorerVoices | Unset):
        limit (int | Unset):  Default: 100.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ExplorerListing
     """


    return sync_detailed(
        client=client,
path=path,
recursive=recursive,
q=q,
state=state,
game=game,
visibility=visibility,
from_=from_,
to=to,
owner_id=owner_id,
person_id=person_id,
voices=voices,
limit=limit,
offset=offset,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,
    recursive: ListLibraryExplorerRecursive | Unset = UNSET,
    q: str | Unset = UNSET,
    state: ListLibraryExplorerState | Unset = UNSET,
    game: str | Unset = UNSET,
    visibility: ListLibraryExplorerVisibility | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    person_id: int | Unset = UNSET,
    voices: ListLibraryExplorerVoices | Unset = UNSET,
    limit: int | Unset = 100,
    offset: int | Unset = 0,

) -> Response[Error | ExplorerListing]:
    """ Explorateur de la Racine vidéo : dossiers et vidéos croisés avec la base, paginés côté serveur (ADR
    0032)

    Args:
        path (str | Unset):
        recursive (ListLibraryExplorerRecursive | Unset):
        q (str | Unset):
        state (ListLibraryExplorerState | Unset):
        game (str | Unset):
        visibility (ListLibraryExplorerVisibility | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):
        owner_id (int | Unset):
        person_id (int | Unset):
        voices (ListLibraryExplorerVoices | Unset):
        limit (int | Unset):  Default: 100.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ExplorerListing]
     """


    kwargs = _get_kwargs(
        path=path,
recursive=recursive,
q=q,
state=state,
game=game,
visibility=visibility,
from_=from_,
to=to,
owner_id=owner_id,
person_id=person_id,
voices=voices,
limit=limit,
offset=offset,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,
    recursive: ListLibraryExplorerRecursive | Unset = UNSET,
    q: str | Unset = UNSET,
    state: ListLibraryExplorerState | Unset = UNSET,
    game: str | Unset = UNSET,
    visibility: ListLibraryExplorerVisibility | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    person_id: int | Unset = UNSET,
    voices: ListLibraryExplorerVoices | Unset = UNSET,
    limit: int | Unset = 100,
    offset: int | Unset = 0,

) -> Error | ExplorerListing | None:
    """ Explorateur de la Racine vidéo : dossiers et vidéos croisés avec la base, paginés côté serveur (ADR
    0032)

    Args:
        path (str | Unset):
        recursive (ListLibraryExplorerRecursive | Unset):
        q (str | Unset):
        state (ListLibraryExplorerState | Unset):
        game (str | Unset):
        visibility (ListLibraryExplorerVisibility | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):
        owner_id (int | Unset):
        person_id (int | Unset):
        voices (ListLibraryExplorerVoices | Unset):
        limit (int | Unset):  Default: 100.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ExplorerListing
     """


    return (await asyncio_detailed(
        client=client,
path=path,
recursive=recursive,
q=q,
state=state,
game=game,
visibility=visibility,
from_=from_,
to=to,
owner_id=owner_id,
person_id=person_id,
voices=voices,
limit=limit,
offset=offset,

    )).parsed
