from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.search_page import SearchPage
from ...models.search_transcripts_mode import SearchTranscriptsMode
from ...models.search_transcripts_status import SearchTranscriptsStatus
from ...models.search_transcripts_visibility import SearchTranscriptsVisibility
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    *,
    q: str,
    mode: SearchTranscriptsMode | Unset = SearchTranscriptsMode.FULL_TEXT,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    visibility: SearchTranscriptsVisibility | Unset = UNSET,
    status: SearchTranscriptsStatus | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    person_id: int | Unset = UNSET,
    person_unknown: bool | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["q"] = q

    json_mode: str | Unset = UNSET
    if not isinstance(mode, Unset):
        json_mode = mode.value

    params["mode"] = json_mode

    params["page"] = page

    params["page_size"] = page_size

    json_visibility: str | Unset = UNSET
    if not isinstance(visibility, Unset):
        json_visibility = visibility.value

    params["visibility"] = json_visibility

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["source_id"] = source_id

    json_from_: str | Unset = UNSET
    if not isinstance(from_, Unset):
        json_from_ = from_.isoformat()
    params["from"] = json_from_

    params["person_id"] = person_id

    params["person_unknown"] = person_unknown

    json_to: str | Unset = UNSET
    if not isinstance(to, Unset):
        json_to = to.isoformat()
    params["to"] = json_to


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/search",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SearchPage:
    if response.status_code == 200:
        response_200 = SearchPage.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SearchPage]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str,
    mode: SearchTranscriptsMode | Unset = SearchTranscriptsMode.FULL_TEXT,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    visibility: SearchTranscriptsVisibility | Unset = UNSET,
    status: SearchTranscriptsStatus | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    person_id: int | Unset = UNSET,
    person_unknown: bool | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> Response[Error | SearchPage]:
    """  Recherche exacte ou plein texte insensible à la casse et aux accents français, suivie en mode plein
    texte d’une correspondance approximative pg_trgm pour les mots d’au moins quatre caractères. Le mode
    hybride fusionne par RRF jusqu’aux 100 meilleurs candidats lexicaux et vectoriels. Les clips privés
    hors propriétaire et administrateur sont exclus avant calcul du total et pagination ; le filtre
    Personne (person_id, person_unknown) s’applique aussi avant total et pagination, au segment, dans
    les trois modes. Chaque résultat porte la Personne identifiée du segment (ou null) et le niveau de
    décision tracé (unknown sans attribution). Aucun appel au worker n’est effectué.

    Args:
        q (str):
        mode (SearchTranscriptsMode | Unset):  Default: SearchTranscriptsMode.FULL_TEXT.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        visibility (SearchTranscriptsVisibility | Unset):
        status (SearchTranscriptsStatus | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        person_id (int | Unset):
        person_unknown (bool | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SearchPage]
     """


    kwargs = _get_kwargs(
        q=q,
mode=mode,
page=page,
page_size=page_size,
visibility=visibility,
status=status,
source_id=source_id,
from_=from_,
person_id=person_id,
person_unknown=person_unknown,
to=to,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    q: str,
    mode: SearchTranscriptsMode | Unset = SearchTranscriptsMode.FULL_TEXT,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    visibility: SearchTranscriptsVisibility | Unset = UNSET,
    status: SearchTranscriptsStatus | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    person_id: int | Unset = UNSET,
    person_unknown: bool | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> Error | SearchPage | None:
    """  Recherche exacte ou plein texte insensible à la casse et aux accents français, suivie en mode plein
    texte d’une correspondance approximative pg_trgm pour les mots d’au moins quatre caractères. Le mode
    hybride fusionne par RRF jusqu’aux 100 meilleurs candidats lexicaux et vectoriels. Les clips privés
    hors propriétaire et administrateur sont exclus avant calcul du total et pagination ; le filtre
    Personne (person_id, person_unknown) s’applique aussi avant total et pagination, au segment, dans
    les trois modes. Chaque résultat porte la Personne identifiée du segment (ou null) et le niveau de
    décision tracé (unknown sans attribution). Aucun appel au worker n’est effectué.

    Args:
        q (str):
        mode (SearchTranscriptsMode | Unset):  Default: SearchTranscriptsMode.FULL_TEXT.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        visibility (SearchTranscriptsVisibility | Unset):
        status (SearchTranscriptsStatus | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        person_id (int | Unset):
        person_unknown (bool | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SearchPage
     """


    return sync_detailed(
        client=client,
q=q,
mode=mode,
page=page,
page_size=page_size,
visibility=visibility,
status=status,
source_id=source_id,
from_=from_,
person_id=person_id,
person_unknown=person_unknown,
to=to,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str,
    mode: SearchTranscriptsMode | Unset = SearchTranscriptsMode.FULL_TEXT,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    visibility: SearchTranscriptsVisibility | Unset = UNSET,
    status: SearchTranscriptsStatus | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    person_id: int | Unset = UNSET,
    person_unknown: bool | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> Response[Error | SearchPage]:
    """  Recherche exacte ou plein texte insensible à la casse et aux accents français, suivie en mode plein
    texte d’une correspondance approximative pg_trgm pour les mots d’au moins quatre caractères. Le mode
    hybride fusionne par RRF jusqu’aux 100 meilleurs candidats lexicaux et vectoriels. Les clips privés
    hors propriétaire et administrateur sont exclus avant calcul du total et pagination ; le filtre
    Personne (person_id, person_unknown) s’applique aussi avant total et pagination, au segment, dans
    les trois modes. Chaque résultat porte la Personne identifiée du segment (ou null) et le niveau de
    décision tracé (unknown sans attribution). Aucun appel au worker n’est effectué.

    Args:
        q (str):
        mode (SearchTranscriptsMode | Unset):  Default: SearchTranscriptsMode.FULL_TEXT.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        visibility (SearchTranscriptsVisibility | Unset):
        status (SearchTranscriptsStatus | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        person_id (int | Unset):
        person_unknown (bool | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SearchPage]
     """


    kwargs = _get_kwargs(
        q=q,
mode=mode,
page=page,
page_size=page_size,
visibility=visibility,
status=status,
source_id=source_id,
from_=from_,
person_id=person_id,
person_unknown=person_unknown,
to=to,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    q: str,
    mode: SearchTranscriptsMode | Unset = SearchTranscriptsMode.FULL_TEXT,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    visibility: SearchTranscriptsVisibility | Unset = UNSET,
    status: SearchTranscriptsStatus | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    person_id: int | Unset = UNSET,
    person_unknown: bool | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> Error | SearchPage | None:
    """  Recherche exacte ou plein texte insensible à la casse et aux accents français, suivie en mode plein
    texte d’une correspondance approximative pg_trgm pour les mots d’au moins quatre caractères. Le mode
    hybride fusionne par RRF jusqu’aux 100 meilleurs candidats lexicaux et vectoriels. Les clips privés
    hors propriétaire et administrateur sont exclus avant calcul du total et pagination ; le filtre
    Personne (person_id, person_unknown) s’applique aussi avant total et pagination, au segment, dans
    les trois modes. Chaque résultat porte la Personne identifiée du segment (ou null) et le niveau de
    décision tracé (unknown sans attribution). Aucun appel au worker n’est effectué.

    Args:
        q (str):
        mode (SearchTranscriptsMode | Unset):  Default: SearchTranscriptsMode.FULL_TEXT.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        visibility (SearchTranscriptsVisibility | Unset):
        status (SearchTranscriptsStatus | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        person_id (int | Unset):
        person_unknown (bool | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SearchPage
     """


    return (await asyncio_detailed(
        client=client,
q=q,
mode=mode,
page=page,
page_size=page_size,
visibility=visibility,
status=status,
source_id=source_id,
from_=from_,
person_id=person_id,
person_unknown=person_unknown,
to=to,

    )).parsed
