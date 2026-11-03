from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.list_people_to_process_filter import ListPeopleToProcessFilter
from ...models.people_to_process import PeopleToProcess
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    filter_: ListPeopleToProcessFilter | Unset = ListPeopleToProcessFilter.ALL,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_filter_: str | Unset = UNSET
    if not isinstance(filter_, Unset):
        json_filter_ = filter_.value

    params["filter"] = json_filter_

    params["page"] = page

    params["page_size"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/people/to-process",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PeopleToProcess:
    if response.status_code == 200:
        response_200 = PeopleToProcess.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PeopleToProcess]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    filter_: ListPeopleToProcessFilter | Unset = ListPeopleToProcessFilter.ALL,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Response[Error | PeopleToProcess]:
    """  Utilisateurs connectés. Vue « À traiter », calculée à la lecture sur les clips visibles : voix sans
    validation humaine ni attribution automatique, hors voix ignorées et Random (ADR 0031). Par clip :
    voix à confirmer (suggestion), ambiguës, inconnues, et nombre de groupes proposés (deux voix ou
    plus) auxquels ses voix appartiennent. Tri : plus de voix d'abord. `total` suit le filtre ; `totals`
    n'en dépend pas (compteurs de l'onglet). Cette lecture n'écrit rien.

    Args:
        filter_ (ListPeopleToProcessFilter | Unset):  Default: ListPeopleToProcessFilter.ALL.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PeopleToProcess]
     """


    kwargs = _get_kwargs(
        filter_=filter_,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    filter_: ListPeopleToProcessFilter | Unset = ListPeopleToProcessFilter.ALL,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Error | PeopleToProcess | None:
    """  Utilisateurs connectés. Vue « À traiter », calculée à la lecture sur les clips visibles : voix sans
    validation humaine ni attribution automatique, hors voix ignorées et Random (ADR 0031). Par clip :
    voix à confirmer (suggestion), ambiguës, inconnues, et nombre de groupes proposés (deux voix ou
    plus) auxquels ses voix appartiennent. Tri : plus de voix d'abord. `total` suit le filtre ; `totals`
    n'en dépend pas (compteurs de l'onglet). Cette lecture n'écrit rien.

    Args:
        filter_ (ListPeopleToProcessFilter | Unset):  Default: ListPeopleToProcessFilter.ALL.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PeopleToProcess
     """


    return sync_detailed(
        client=client,
filter_=filter_,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    filter_: ListPeopleToProcessFilter | Unset = ListPeopleToProcessFilter.ALL,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Response[Error | PeopleToProcess]:
    """  Utilisateurs connectés. Vue « À traiter », calculée à la lecture sur les clips visibles : voix sans
    validation humaine ni attribution automatique, hors voix ignorées et Random (ADR 0031). Par clip :
    voix à confirmer (suggestion), ambiguës, inconnues, et nombre de groupes proposés (deux voix ou
    plus) auxquels ses voix appartiennent. Tri : plus de voix d'abord. `total` suit le filtre ; `totals`
    n'en dépend pas (compteurs de l'onglet). Cette lecture n'écrit rien.

    Args:
        filter_ (ListPeopleToProcessFilter | Unset):  Default: ListPeopleToProcessFilter.ALL.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PeopleToProcess]
     """


    kwargs = _get_kwargs(
        filter_=filter_,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    filter_: ListPeopleToProcessFilter | Unset = ListPeopleToProcessFilter.ALL,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Error | PeopleToProcess | None:
    """  Utilisateurs connectés. Vue « À traiter », calculée à la lecture sur les clips visibles : voix sans
    validation humaine ni attribution automatique, hors voix ignorées et Random (ADR 0031). Par clip :
    voix à confirmer (suggestion), ambiguës, inconnues, et nombre de groupes proposés (deux voix ou
    plus) auxquels ses voix appartiennent. Tri : plus de voix d'abord. `total` suit le filtre ; `totals`
    n'en dépend pas (compteurs de l'onglet). Cette lecture n'écrit rien.

    Args:
        filter_ (ListPeopleToProcessFilter | Unset):  Default: ListPeopleToProcessFilter.ALL.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PeopleToProcess
     """


    return (await asyncio_detailed(
        client=client,
filter_=filter_,
page=page,
page_size=page_size,

    )).parsed
