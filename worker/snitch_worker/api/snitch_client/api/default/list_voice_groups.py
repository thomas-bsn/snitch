from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.list_voice_groups_sort import ListVoiceGroupsSort
from ...models.voice_group_list import VoiceGroupList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    sort: ListVoiceGroupsSort | Unset = ListVoiceGroupsSort.CLIPS,
    proposed: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    voice: int | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_sort: str | Unset = UNSET
    if not isinstance(sort, Unset):
        json_sort = sort.value

    params["sort"] = json_sort

    params["proposed"] = proposed

    params["page"] = page

    params["page_size"] = page_size

    params["voice"] = voice


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/voice-groups",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceGroupList:
    if response.status_code == 200:
        response_200 = VoiceGroupList.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceGroupList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    sort: ListVoiceGroupsSort | Unset = ListVoiceGroupsSort.CLIPS,
    proposed: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    voice: int | Unset = UNSET,

) -> Response[Error | VoiceGroupList]:
    """  Utilisateurs connectés. Une voix inconnue (non validée par un humain, non attribuée automatiquement)
    forme un groupe à elle seule tant qu'un lien de proximité proposé ne la joint pas à une autre. Les
    groupes ne sont jamais confirmés automatiquement. Nombres de voix et de clips calculés en base sur
    les seuls clips visibles ; tri par nombre de clips.

    Args:
        sort (ListVoiceGroupsSort | Unset):  Default: ListVoiceGroupsSort.CLIPS.
        proposed (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        voice (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceGroupList]
     """


    kwargs = _get_kwargs(
        sort=sort,
proposed=proposed,
page=page,
page_size=page_size,
voice=voice,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    sort: ListVoiceGroupsSort | Unset = ListVoiceGroupsSort.CLIPS,
    proposed: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    voice: int | Unset = UNSET,

) -> Error | VoiceGroupList | None:
    """  Utilisateurs connectés. Une voix inconnue (non validée par un humain, non attribuée automatiquement)
    forme un groupe à elle seule tant qu'un lien de proximité proposé ne la joint pas à une autre. Les
    groupes ne sont jamais confirmés automatiquement. Nombres de voix et de clips calculés en base sur
    les seuls clips visibles ; tri par nombre de clips.

    Args:
        sort (ListVoiceGroupsSort | Unset):  Default: ListVoiceGroupsSort.CLIPS.
        proposed (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        voice (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceGroupList
     """


    return sync_detailed(
        client=client,
sort=sort,
proposed=proposed,
page=page,
page_size=page_size,
voice=voice,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    sort: ListVoiceGroupsSort | Unset = ListVoiceGroupsSort.CLIPS,
    proposed: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    voice: int | Unset = UNSET,

) -> Response[Error | VoiceGroupList]:
    """  Utilisateurs connectés. Une voix inconnue (non validée par un humain, non attribuée automatiquement)
    forme un groupe à elle seule tant qu'un lien de proximité proposé ne la joint pas à une autre. Les
    groupes ne sont jamais confirmés automatiquement. Nombres de voix et de clips calculés en base sur
    les seuls clips visibles ; tri par nombre de clips.

    Args:
        sort (ListVoiceGroupsSort | Unset):  Default: ListVoiceGroupsSort.CLIPS.
        proposed (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        voice (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceGroupList]
     """


    kwargs = _get_kwargs(
        sort=sort,
proposed=proposed,
page=page,
page_size=page_size,
voice=voice,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    sort: ListVoiceGroupsSort | Unset = ListVoiceGroupsSort.CLIPS,
    proposed: bool | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,
    voice: int | Unset = UNSET,

) -> Error | VoiceGroupList | None:
    """  Utilisateurs connectés. Une voix inconnue (non validée par un humain, non attribuée automatiquement)
    forme un groupe à elle seule tant qu'un lien de proximité proposé ne la joint pas à une autre. Les
    groupes ne sont jamais confirmés automatiquement. Nombres de voix et de clips calculés en base sur
    les seuls clips visibles ; tri par nombre de clips.

    Args:
        sort (ListVoiceGroupsSort | Unset):  Default: ListVoiceGroupsSort.CLIPS.
        proposed (bool | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.
        voice (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceGroupList
     """


    return (await asyncio_detailed(
        client=client,
sort=sort,
proposed=proposed,
page=page,
page_size=page_size,
voice=voice,

    )).parsed
